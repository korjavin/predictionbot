package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// DefaultDisputeDelay is the default time to wait before auto-finalizing a resolved market
const DefaultDisputeDelay = 24 * time.Hour

// MarketWorker handles background tasks for markets
type MarketWorker struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	ticker              *time.Ticker
	disputeDelay        time.Duration
	notificationService *NotificationService
}

// NewMarketWorker creates a new market worker
func NewMarketWorker() *MarketWorker {
	ctx, cancel := context.WithCancel(context.Background())

	// Get configurable dispute delay from environment (for testing, can be set to 1 minute)
	disputeDelayStr := os.Getenv("DISPUTE_DELAY_MINUTES")
	disputeDelay := DefaultDisputeDelay
	if disputeDelayStr != "" {
		if minutes, err := strconv.Atoi(disputeDelayStr); err == nil && minutes > 0 {
			disputeDelay = time.Duration(minutes) * time.Minute
			logger.Debug(0, "market_worker_config", fmt.Sprintf("dispute_delay=%d minutes", minutes))
		}
	}

	return &MarketWorker{
		ctx:          ctx,
		cancel:       cancel,
		ticker:       time.NewTicker(1 * time.Minute),
		disputeDelay: disputeDelay,
	}
}

// Start begins the background worker
func (w *MarketWorker) Start() {
	logger.Debug(0, "market_worker_started", fmt.Sprintf("interval=1m dispute_delay=%v", w.disputeDelay))

	// Run immediately on start
	w.lockExpiredMarkets()
	w.autoFinalizeResolvedMarkets()

	// Then run on ticker
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.lockExpiredMarkets()
				w.autoFinalizeResolvedMarkets()
			case <-w.ctx.Done():
				logger.Debug(0, "market_worker_stopped", "")
				return
			}
		}
	}()
}

// Stop stops the background worker
func (w *MarketWorker) Stop() {
	w.ticker.Stop()
	w.cancel()
}

// SetNotificationService sets the notification service for payout notifications
func (w *MarketWorker) SetNotificationService(ns *NotificationService) {
	w.notificationService = ns
}

// lockExpiredMarkets finds and locks all expired active markets
func (w *MarketWorker) lockExpiredMarkets() {
	db := storage.DB()
	if db == nil {
		logger.Debug(0, "market_worker_no_db", "")
		return
	}

	// First, get the locked markets with their details before updating
	lockedMarkets, err := w.getExpiredMarkets()
	if err != nil {
		logger.Debug(0, "market_worker_query_failed", fmt.Sprintf("error=%s", err.Error()))
		return
	}

	if len(lockedMarkets) == 0 {
		return // No markets to lock
	}

	// Update markets to LOCKED status
	marketIDs := make([]int64, len(lockedMarkets))
	for i, m := range lockedMarkets {
		marketIDs[i] = m.ID
	}

	placeholders := "?"
	for i := 1; i < len(marketIDs); i++ {
		placeholders += ", ?"
	}

	query := fmt.Sprintf(`
		UPDATE markets
		SET status = 'LOCKED'
		WHERE id IN (%s)
	`, placeholders)

	args := make([]interface{}, len(marketIDs))
	for i, id := range marketIDs {
		args[i] = id
	}

	_, err = db.ExecContext(w.ctx, query, args...)
	if err != nil {
		logger.Debug(0, "market_worker_lock_failed", fmt.Sprintf("error=%s", err.Error()))
		return
	}

	logger.Debug(0, "market_worker_locked_markets", fmt.Sprintf("count=%d", len(lockedMarkets)))

	// Send deadline notifications to market creators
	if w.notificationService != nil {
		for _, market := range lockedMarkets {
			w.notificationService.NotifyMarketCreatorDeadline(market)
		}
	}
}

// getExpiredMarkets returns markets that have expired but are still active
func (w *MarketWorker) getExpiredMarkets() ([]*storage.Market, error) {
	db := storage.DB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	rows, err := db.QueryContext(w.ctx, `
		SELECT id, creator_id, question, image_url, status, outcome, resolved_at, expires_at, created_at
		FROM markets
		WHERE status = 'ACTIVE'
		AND expires_at < CURRENT_TIMESTAMP
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []*storage.Market
	for rows.Next() {
		var market storage.Market
		var imageURL, outcome sql.NullString
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&market.ID,
			&market.CreatorID,
			&market.Question,
			&imageURL,
			&market.Status,
			&outcome,
			&resolvedAt,
			&market.ExpiresAt,
			&market.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if imageURL.Valid {
			market.ImageURL = imageURL.String
		}
		if outcome.Valid {
			market.Outcome = outcome.String
		}
		if resolvedAt.Valid {
			market.ResolvedAt = resolvedAt.Time
		}

		markets = append(markets, &market)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return markets, nil
}

// autoFinalizeResolvedMarkets finds resolved markets past the dispute period and finalizes them
func (w *MarketWorker) autoFinalizeResolvedMarkets() {
	db := storage.DB()
	if db == nil {
		logger.Debug(0, "market_worker_no_db", "")
		return
	}

	// Get markets that are resolved and past the dispute period
	marketIDs, err := storage.GetMarketsPendingFinalization(w.disputeDelay)
	if err != nil {
		logger.Debug(0, "market_worker_pending_query_failed", fmt.Sprintf("error=%s", err.Error()))
		return
	}

	if len(marketIDs) == 0 {
		return
	}

	logger.Debug(0, "market_worker_auto_finalize", fmt.Sprintf("count=%d", len(marketIDs)))

	payoutService := NewPayoutService()
	if w.notificationService != nil {
		payoutService.SetNotificationService(w.notificationService)
	}

	// Finalize each market
	for _, marketID := range marketIDs {
		payoutsProcessed, err := payoutService.FinalizeMarket(w.ctx, marketID, "")
		if err != nil {
			logger.Debug(0, "market_worker_finalize_failed", fmt.Sprintf("market_id=%d error=%s", marketID, err.Error()))
			continue
		}
		logger.Debug(0, "market_worker_finalized", fmt.Sprintf("market_id=%d payouts=%d", marketID, payoutsProcessed))
	}
}
