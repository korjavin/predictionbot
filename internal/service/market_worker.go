package service

import (
	"context"
	"fmt"
	"time"

	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// MarketWorker handles background tasks for markets
type MarketWorker struct {
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// NewMarketWorker creates a new market worker
func NewMarketWorker() *MarketWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &MarketWorker{
		ctx:    ctx,
		cancel: cancel,
		ticker: time.NewTicker(1 * time.Minute),
	}
}

// Start begins the background worker
func (w *MarketWorker) Start() {
	logger.Debug(0, "market_worker_started", "interval=1m")

	// Run immediately on start
	w.lockExpiredMarkets()

	// Then run on ticker
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.lockExpiredMarkets()
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

// lockExpiredMarkets finds and locks all expired active markets
func (w *MarketWorker) lockExpiredMarkets() {
	db := storage.DB()
	if db == nil {
		logger.Debug(0, "market_worker_no_db", "")
		return
	}

	// Query markets where status = 'ACTIVE' AND expires_at < NOW
	result, err := db.ExecContext(w.ctx, `
		UPDATE markets
		SET status = 'LOCKED'
		WHERE status = 'ACTIVE'
		AND expires_at < CURRENT_TIMESTAMP
	`)
	if err != nil {
		logger.Debug(0, "market_worker_lock_failed", fmt.Sprintf("error=%s", err.Error()))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logger.Debug(0, "market_worker_rows_error", fmt.Sprintf("error=%s", err.Error()))
		return
	}

	if rowsAffected > 0 {
		logger.Debug(0, "market_worker_locked_markets", fmt.Sprintf("count=%d", rowsAffected))
	}
}
