package service

import (
	"context"
	"database/sql"
	"fmt"

	"predictionbot/internal/logger"
	"predictionbot/internal/storage"
)

// PayoutService handles market resolution and payouts
type PayoutService struct{}

// NewPayoutService creates a new payout service
func NewPayoutService() *PayoutService {
	return &PayoutService{}
}

// ResolveMarket resolves a market (Creator Action)
// This sets the market status to RESOLVED and stores the outcome
// Money is NOT distributed yet - it waits for the dispute period
func (s *PayoutService) ResolveMarket(ctx context.Context, marketID, creatorID int64, outcome string) error {
	// Validate outcome
	if outcome != "YES" && outcome != "NO" {
		return fmt.Errorf("invalid outcome: must be 'YES' or 'NO'")
	}

	db := storage.DB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Validate that the market exists and the user is the creator
	var actualCreatorID int64
	var currentStatus string
	err := db.QueryRowContext(ctx, `
		SELECT creator_id, status
		FROM markets
		WHERE id = ?
	`, marketID).Scan(&actualCreatorID, &currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("market not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get market: %w", err)
	}

	// Only creator can resolve
	if actualCreatorID != creatorID {
		return fmt.Errorf("only the market creator can resolve this market")
	}

	// Market must be LOCKED
	if currentStatus != string(storage.MarketStatusLocked) {
		return fmt.Errorf("market cannot be resolved: status is %s", currentStatus)
	}

	// Update market status to RESOLVED with outcome
	err = storage.UpdateMarketStatus(marketID, storage.MarketStatusResolved, outcome)
	if err != nil {
		return fmt.Errorf("failed to resolve market: %w", err)
	}

	logger.Debug(creatorID, "market_resolved", fmt.Sprintf("market_id=%d outcome=%s", marketID, outcome))

	return nil
}

// RaiseDispute raises a dispute on a resolved market (User Action)
// This sets the market status to DISPUTED and stops auto-finalization
func (s *PayoutService) RaiseDispute(ctx context.Context, marketID, userID int64) error {
	db := storage.DB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Validate that the market exists and is in RESOLVED status
	var currentStatus string
	err := db.QueryRowContext(ctx, `
		SELECT status
		FROM markets
		WHERE id = ?
	`, marketID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("market not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get market: %w", err)
	}

	// Market must be in RESOLVED status to be disputed
	if currentStatus != string(storage.MarketStatusResolved) {
		return fmt.Errorf("market cannot be disputed: status is %s", currentStatus)
	}

	// Update market status to DISPUTED
	err = storage.UpdateMarketStatus(marketID, storage.MarketStatusDisputed, "")
	if err != nil {
		return fmt.Errorf("failed to dispute market: %w", err)
	}

	logger.Debug(userID, "market_disputed", fmt.Sprintf("market_id=%d", marketID))

	return nil
}

// FinalizeMarket finalizes a market and distributes payouts
// This can be called by:
// - Admin (with forceOutcome) to resolve disputed markets
// - System (auto-finalization) to resolve markets after dispute period
func (s *PayoutService) FinalizeMarket(ctx context.Context, marketID int64, forceOutcome string) (int, error) {
	db := storage.DB()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	// Get market details
	var marketStatus string
	var storedOutcome string
	err := db.QueryRowContext(ctx, `
		SELECT status, outcome
		FROM markets
		WHERE id = ?
	`, marketID).Scan(&marketStatus, &storedOutcome)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("market not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get market: %w", err)
	}

	// Market must be RESOLVED or DISPUTED
	if marketStatus != string(storage.MarketStatusResolved) && marketStatus != string(storage.MarketStatusDisputed) {
		return 0, fmt.Errorf("market cannot be finalized: status is %s", marketStatus)
	}

	// Use forceOutcome if provided (admin case), otherwise use stored outcome
	outcome := storedOutcome
	if forceOutcome != "" {
		if forceOutcome != "YES" && forceOutcome != "NO" {
			return 0, fmt.Errorf("invalid outcome: must be 'YES' or 'NO'")
		}
		outcome = forceOutcome
	}

	// Begin transaction with serializable isolation
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all bets for this market
	rows, err := tx.QueryContext(ctx, `
		SELECT id, user_id, outcome, amount
		FROM bets
		WHERE market_id = ?
	`, marketID)
	if err != nil {
		return 0, fmt.Errorf("failed to get bets: %w", err)
	}
	defer rows.Close()

	type bet struct {
		ID      int64
		UserID  int64
		Outcome string
		Amount  int64
	}

	var bets []bet
	totalPool := int64(0)
	winningPool := int64(0)

	for rows.Next() {
		var b bet
		err := rows.Scan(&b.ID, &b.UserID, &b.Outcome, &b.Amount)
		if err != nil {
			return 0, fmt.Errorf("failed to scan bet: %w", err)
		}
		bets = append(bets, b)
		totalPool += b.Amount
		if b.Outcome == outcome {
			winningPool += b.Amount
		}
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating bets: %w", err)
	}

	logger.Debug(0, "market_finalization_started", fmt.Sprintf("market_id=%d outcome=%s total_pool=%d winning_pool=%d", marketID, outcome, totalPool, winningPool))

	payoutsProcessed := 0

	// Edge case: Nobody bet on the winning outcome (WinningPool == 0)
	// Refund everyone who bet
	if winningPool == 0 {
		logger.Debug(0, "market_finalization_no_winners", fmt.Sprintf("market_id=%d refunding_all", marketID))

		for _, b := range bets {
			// Refund the bet amount
			_, err = tx.ExecContext(ctx, `
				UPDATE users
				SET balance = balance + ?
				WHERE id = ?
			`, b.Amount, b.UserID)
			if err != nil {
				return 0, fmt.Errorf("failed to refund user %d: %w", b.UserID, err)
			}

			// Log refund transaction
			_, err = tx.ExecContext(ctx, `
				INSERT INTO transactions (user_id, amount, source_type, description)
				VALUES (?, ?, 'REFUND', ?)
			`, b.UserID, b.Amount, fmt.Sprintf("Refund for bet #%d on market #%d (no winning bets)", b.ID, marketID))
			if err != nil {
				return 0, fmt.Errorf("failed to log refund transaction: %w", err)
			}

			payoutsProcessed++
		}
	} else {
		// Calculate and distribute winnings using parimutuel formula
		// Payout = (UserBet * TotalPool) / WinningPool
		for _, b := range bets {
			if b.Outcome == outcome {
				// Calculate payout using integer arithmetic
				payout := (b.Amount * totalPool) / winningPool

				// Update user balance
				_, err = tx.ExecContext(ctx, `
					UPDATE users
					SET balance = balance + ?
					WHERE id = ?
				`, payout, b.UserID)
				if err != nil {
					return 0, fmt.Errorf("failed to update user %d balance: %w", b.UserID, err)
				}

				// Log win payout transaction
				netProfit := payout - b.Amount
				_, err = tx.ExecContext(ctx, `
					INSERT INTO transactions (user_id, amount, source_type, description)
					VALUES (?, ?, 'WIN_PAYOUT', ?)
				`, b.UserID, payout, fmt.Sprintf("Win payout for bet #%d on market #%d (bet: %d, payout: %d, profit: %d)", b.ID, marketID, b.Amount, payout, netProfit))
				if err != nil {
					return 0, fmt.Errorf("failed to log win transaction: %w", err)
				}

				payoutsProcessed++
				logger.Debug(b.UserID, "payout_processed", fmt.Sprintf("bet_id=%d market_id=%d bet_amount=%d payout=%d profit=%d", b.ID, marketID, b.Amount, payout, netProfit))
			}
		}
	}

	// Update market status to FINALIZED with outcome and resolved_at
	_, err = tx.ExecContext(ctx, `
		UPDATE markets
		SET status = 'FINALIZED', outcome = ?, resolved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, outcome, marketID)
	if err != nil {
		return 0, fmt.Errorf("failed to finalize market: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Debug(0, "market_finalization_completed", fmt.Sprintf("market_id=%d outcome=%s payouts=%d", marketID, outcome, payoutsProcessed))

	return payoutsProcessed, nil
}
