package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	// WelcomeBonusAmount is the welcome bonus amount
	WelcomeBonusAmount int64 = 1000
	// BailoutAmount is the bailout amount
	BailoutAmount int64 = 500
	// BailoutCooldown is the cooldown period for bailouts (24 hours)
	BailoutCooldown = 24 * time.Hour
	// BailoutBalanceThreshold is the minimum balance to be eligible for bailout
	BailoutBalanceThreshold int64 = 1
)

var db *sql.DB

// InitDB initializes the SQLite database connection with WAL mode
func InitDB(dbPath string) error {
	var err error

	// For in-memory databases, use the path directly
	// Otherwise ensure the directory exists
	if dbPath != ":memory:" {
		_, err = filepath.Abs(dbPath)
		if err != nil {
			return err
		}
	}

	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return err
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return err
	}

	return nil
}

// DB returns the database connection
func DB() *sql.DB {
	return db
}

// runMigrations creates the necessary tables
func runMigrations() error {
	usersTable := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			telegram_id INTEGER UNIQUE NOT NULL,
			username TEXT,
			first_name TEXT NOT NULL,
			balance INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	transactionsTable := `
		CREATE TABLE IF NOT EXISTS transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			amount INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	marketsTable := `
		CREATE TABLE IF NOT EXISTS markets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			creator_id INTEGER NOT NULL,
			question TEXT NOT NULL,
			image_url TEXT,
			status TEXT NOT NULL DEFAULT 'ACTIVE',
			outcome TEXT,
			resolved_at DATETIME,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (creator_id) REFERENCES users(id)
		)
	`

	betsTable := `
		CREATE TABLE IF NOT EXISTS bets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			market_id INTEGER NOT NULL,
			outcome TEXT NOT NULL CHECK (outcome IN ('YES', 'NO')),
			amount INTEGER NOT NULL,
			placed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (market_id) REFERENCES markets(id)
		)
	`

	// Create indexes for better query performance
	createIndexes := `
		CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
		CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
		CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);
		CREATE INDEX IF NOT EXISTS idx_markets_created_at ON markets(created_at);
		CREATE INDEX IF NOT EXISTS idx_bets_user_market ON bets(user_id, market_id);
		CREATE INDEX IF NOT EXISTS idx_bets_market ON bets(market_id);
		CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);
	`

	_, err := db.Exec(usersTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(transactionsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(marketsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(betsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createIndexes)
	if err != nil {
		return err
	}

	// Migration: Add outcome and resolved_at columns if they don't exist
	// SQLite's ALTER TABLE ADD COLUMN is idempotent-ish (won't fail if column exists in newer versions)
	// But we'll check first to be safe
	var outcomeExists int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('markets') WHERE name='outcome'").Scan(&outcomeExists)
	if err != nil {
		return err
	}
	if outcomeExists == 0 {
		_, err = db.Exec("ALTER TABLE markets ADD COLUMN outcome TEXT")
		if err != nil {
			return err
		}
	}

	var resolvedAtExists int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('markets') WHERE name='resolved_at'").Scan(&resolvedAtExists)
	if err != nil {
		return err
	}
	if resolvedAtExists == 0 {
		_, err = db.Exec("ALTER TABLE markets ADD COLUMN resolved_at DATETIME")
		if err != nil {
			return err
		}
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// GetUserByTelegramID retrieves a user by their Telegram ID
func GetUserByTelegramID(telegramID int64) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT id, telegram_id, username, first_name, balance, created_at, updated_at
		FROM users
		WHERE telegram_id = ?
	`, telegramID).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by telegram_id: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by their internal ID
func GetUserByID(id int64) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT id, telegram_id, username, first_name, balance, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(
		&user.ID,
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.Balance,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}

// CreateUser creates a new user with the given Telegram info and welcome bonus
func CreateUser(telegramID int64, username, firstName string) (*User, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert user with initial balance
	result, err := tx.Exec(`
		INSERT INTO users (telegram_id, username, first_name, balance)
		VALUES (?, ?, ?, ?)
	`, telegramID, username, firstName, WelcomeBonusAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Create welcome bonus transaction
	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, amount, source_type, description)
		VALUES (?, ?, 'WELCOME_BONUS', 'Welcome bonus for joining!')
	`, userID, WelcomeBonusAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to insert welcome bonus transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return the created user
	return GetUserByTelegramID(telegramID)
}

// CreateMarket creates a new market
func CreateMarket(creatorID int64, question string, expiresAt time.Time) (*Market, error) {
	result, err := db.Exec(`
		INSERT INTO markets (creator_id, question, status, expires_at)
		VALUES (?, ?, 'ACTIVE', ?)
	`, creatorID, question, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert market: %w", err)
	}

	marketID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Fetch and return the created market
	return GetMarketByID(marketID)
}

// GetMarketByID retrieves a market by its ID
func GetMarketByID(id int64) (*Market, error) {
	var market Market
	var imageURL sql.NullString
	var outcome sql.NullString
	var resolvedAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, creator_id, question, image_url, status, outcome, resolved_at, expires_at, created_at
		FROM markets
		WHERE id = ?
	`, id).Scan(
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
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get market by id: %w", err)
	}

	// Handle NULL values
	if imageURL.Valid {
		market.ImageURL = imageURL.String
	}
	if outcome.Valid {
		market.Outcome = outcome.String
	}
	if resolvedAt.Valid {
		market.ResolvedAt = resolvedAt.Time
	}

	return &market, nil
}

// ListActiveMarkets retrieves all active markets ordered by creation date (newest first)
func ListActiveMarkets() ([]Market, error) {
	rows, err := db.Query(`
		SELECT id, creator_id, question, image_url, status, expires_at, created_at
		FROM markets
		WHERE status = 'ACTIVE'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query active markets: %w", err)
	}
	defer rows.Close()

	var markets []Market
	for rows.Next() {
		var market Market
		var imageURL sql.NullString
		err := rows.Scan(
			&market.ID,
			&market.CreatorID,
			&market.Question,
			&imageURL,
			&market.Status,
			&market.ExpiresAt,
			&market.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market: %w", err)
		}

		if imageURL.Valid {
			market.ImageURL = imageURL.String
		}

		markets = append(markets, market)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating markets: %w", err)
	}

	return markets, nil
}

// MarketWithCreator represents a market with creator name for API responses
type MarketWithCreator struct {
	ID          int64  `json:"id"`
	Question    string `json:"question"`
	CreatorName string `json:"creator_name"`
	ExpiresAt   string `json:"expires_at"`
	PoolYes     int64  `json:"pool_yes"`
	PoolNo      int64  `json:"pool_no"`
}

// ListActiveMarketsWithCreator returns active markets with creator names
func ListActiveMarketsWithCreator() ([]MarketWithCreator, error) {
	rows, err := db.Query(`
		SELECT m.id, m.question, COALESCE(u.first_name, 'Unknown'),
		       m.expires_at, 0, 0
		FROM markets m
		LEFT JOIN users u ON m.creator_id = u.id
		WHERE m.status = 'ACTIVE'
		ORDER BY m.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query active markets: %w", err)
	}
	defer rows.Close()

	var markets []MarketWithCreator
	for rows.Next() {
		var market MarketWithCreator
		err := rows.Scan(
			&market.ID,
			&market.Question,
			&market.CreatorName,
			&market.ExpiresAt,
			&market.PoolYes,
			&market.PoolNo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market: %w", err)
		}
		markets = append(markets, market)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating markets: %w", err)
	}

	return markets, nil
}

// PlaceBet places a bet on a market with ACID transaction
func PlaceBet(ctx context.Context, userID, marketID int64, outcome string, amount int64) error {
	// Validate outcome
	if outcome != string(OutcomeYes) && outcome != string(OutcomeNo) {
		return fmt.Errorf("invalid outcome: must be 'YES' or 'NO'")
	}

	// Validate amount
	if amount <= 0 {
		return fmt.Errorf("invalid amount: must be greater than 0")
	}

	// Begin immediate transaction for atomicity
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check user balance
	var userBalance int64
	err = tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = ?`, userID).Scan(&userBalance)
	if err == sql.ErrNoRows {
		return fmt.Errorf("user not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	if userBalance < amount {
		return fmt.Errorf("insufficient funds: have %d, need %d", userBalance, amount)
	}

	// Check market exists and is active
	var marketStatus string
	var expiresAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT status, expires_at FROM markets WHERE id = ?`, marketID).Scan(&marketStatus, &expiresAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("market not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get market: %w", err)
	}

	if marketStatus != string(MarketStatusActive) {
		return fmt.Errorf("market is not active: status is %s", marketStatus)
	}

	if time.Now().After(expiresAt) {
		return fmt.Errorf("market has expired")
	}

	// Update user balance
	_, err = tx.ExecContext(ctx, `UPDATE users SET balance = balance - ? WHERE id = ?`, amount, userID)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Insert bet record
	result, err := tx.ExecContext(ctx, `
		INSERT INTO bets (user_id, market_id, outcome, amount)
		VALUES (?, ?, ?, ?)
	`, userID, marketID, outcome, amount)
	if err != nil {
		return fmt.Errorf("failed to insert bet: %w", err)
	}

	betID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get bet id: %w", err)
	}

	// Log the transaction
	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (user_id, amount, source_type, description)
		VALUES (?, ?, 'BET_PLACED', ?)
	`, userID, -amount, fmt.Sprintf("Bet #%d on market #%d (%s)", betID, marketID, outcome))
	if err != nil {
		return fmt.Errorf("failed to log transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetPoolTotals calculates the total pool amounts for a market
func GetPoolTotals(marketID int64) (poolYes, poolNo int64, err error) {
	err = db.QueryRow(`
		SELECT COALESCE(SUM(CASE WHEN outcome = 'YES' THEN amount ELSE 0 END), 0) as pool_yes,
		       COALESCE(SUM(CASE WHEN outcome = 'NO' THEN amount ELSE 0 END), 0) as pool_no
		FROM bets
		WHERE market_id = ?
	`, marketID).Scan(&poolYes, &poolNo)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get pool totals: %w", err)
	}
	return poolYes, poolNo, nil
}

// GetMarketWithPools returns a market with pool totals populated
func GetMarketWithPools(marketID int64) (*MarketWithCreator, error) {
	var market MarketWithCreator
	err := db.QueryRow(`
		SELECT m.id, m.question, COALESCE(u.first_name, 'Unknown'),
		       m.expires_at, 0, 0
		FROM markets m
		LEFT JOIN users u ON m.creator_id = u.id
		WHERE m.id = ?
	`, marketID).Scan(
		&market.ID,
		&market.Question,
		&market.CreatorName,
		&market.ExpiresAt,
		&market.PoolYes,
		&market.PoolNo,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get market: %w", err)
	}

	// Get pool totals
	market.PoolYes, market.PoolNo, err = GetPoolTotals(marketID)
	if err != nil {
		return nil, err
	}

	return &market, nil
}

// UpdateMarketStatus updates the status and optionally the outcome of a market
func UpdateMarketStatus(marketID int64, status MarketStatus, outcome string) error {
	var query string
	var args []interface{}

	if outcome != "" {
		query = `UPDATE markets SET status = ?, outcome = ?, resolved_at = CURRENT_TIMESTAMP WHERE id = ?`
		args = []interface{}{status, outcome, marketID}
	} else {
		query = `UPDATE markets SET status = ? WHERE id = ?`
		args = []interface{}{status, marketID}
	}

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update market status: %w", err)
	}
	return nil
}

// GetMarketsPendingFinalization returns markets that are resolved and ready for auto-finalization
// These are markets where resolved_at is older than the threshold duration
func GetMarketsPendingFinalization(threshold time.Duration) ([]int64, error) {
	rows, err := db.Query(`
		SELECT id FROM markets
		WHERE status = 'RESOLVED'
		AND resolved_at < datetime('now', '-' || ? || ' seconds')
	`, int64(threshold.Seconds()))
	if err != nil {
		return nil, fmt.Errorf("failed to query pending markets: %w", err)
	}
	defer rows.Close()

	var marketIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan market id: %w", err)
		}
		marketIDs = append(marketIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating markets: %w", err)
	}

	return marketIDs, nil
}

// BetStatus represents the status of a bet
type BetStatus string

const (
	BetStatusPending  BetStatus = "PENDING"
	BetStatusWon      BetStatus = "WON"
	BetStatusLost     BetStatus = "LOST"
	BetStatusRefunded BetStatus = "REFUNDED"
)

// BetHistoryItem represents a single bet in the user's history
type BetHistoryItem struct {
	ID            int64     `json:"id"`
	MarketID      int64     `json:"market_id"`
	Question      string    `json:"question"`
	OutcomeChosen string    `json:"outcome_chosen"`
	Amount        int64     `json:"amount"`
	Status        BetStatus `json:"status"`
	Payout        int64     `json:"payout,omitempty"`
	PlacedAt      string    `json:"placed_at"`
}

// GetUserBets returns all bets for a user with computed status based on market outcome
func GetUserBets(userID int64) ([]BetHistoryItem, error) {
	rows, err := db.Query(`
		SELECT b.id, b.market_id, m.question, b.outcome, b.amount, b.placed_at,
		       m.status as market_status, m.outcome as market_outcome
		FROM bets b
		JOIN markets m ON b.market_id = m.id
		WHERE b.user_id = ?
		ORDER BY b.placed_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user bets: %w", err)
	}
	defer rows.Close()

	var bets []BetHistoryItem
	for rows.Next() {
		var b BetHistoryItem
		var marketStatus, marketOutcome sql.NullString
		var placedAt time.Time

		err := rows.Scan(&b.ID, &b.MarketID, &b.Question, &b.OutcomeChosen, &b.Amount, &placedAt, &marketStatus, &marketOutcome)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bet: %w", err)
		}

		b.PlacedAt = placedAt.Format("2006-01-02T15:04:05Z07:00")

		// Determine bet status based on market status and outcome
		b.Status = computeBetStatus(marketStatus.String, marketOutcome.String, b.OutcomeChosen)

		// Calculate payout for won bets
		if b.Status == BetStatusWon {
			// Get the payout amount from transactions
			var payout int64
			err = db.QueryRow(`
				SELECT amount
				FROM transactions
				WHERE user_id = ? AND source_type = 'WIN_PAYOUT'
				AND description LIKE ?
			`, userID, fmt.Sprintf("%%bet #%% on market #%d%%", b.MarketID)).Scan(&payout)
			if err == nil && payout > 0 {
				b.Payout = payout
			}
		}

		bets = append(bets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bets: %w", err)
	}

	return bets, nil
}

// computeBetStatus determines the status of a bet based on market state
func computeBetStatus(marketStatus, marketOutcome, betOutcome string) BetStatus {
	// Active markets are pending
	if marketStatus == string(MarketStatusActive) || marketStatus == string(MarketStatusLocked) {
		return BetStatusPending
	}

	// Refunded if market was never resolved (edge case)
	if marketStatus == "" && marketOutcome == "" {
		return BetStatusRefunded
	}

	// Market is resolved/finalized
	if marketOutcome == "" {
		return BetStatusPending
	}

	// If bet outcome matches market outcome, it's a win
	if betOutcome == marketOutcome {
		return BetStatusWon
	}

	// Otherwise it's a loss
	return BetStatusLost
}

// UserStats represents user statistics
type UserStats struct {
	TotalBets  int     `json:"total_bets"`
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	WinRate    float64 `json:"win_rate"`
	TotalWager int64   `json:"total_wager"`
	TotalWins  int64   `json:"total_wins"`
}

// GetUserStats returns statistics for a user
func GetUserStats(userID int64) (*UserStats, error) {
	stats := &UserStats{}

	// Get total bets count
	err := db.QueryRow(`
		SELECT COUNT(*) FROM bets WHERE user_id = ?
	`, userID).Scan(&stats.TotalBets)
	if err != nil {
		return nil, fmt.Errorf("failed to get total bets: %w", err)
	}

	// Get wins count
	err = db.QueryRow(`
		SELECT COUNT(*) FROM bets b
		JOIN markets m ON b.market_id = m.id
		WHERE b.user_id = ? AND m.status = 'FINALIZED' AND m.outcome = b.outcome
	`, userID).Scan(&stats.Wins)
	if err != nil {
		return nil, fmt.Errorf("failed to get wins count: %w", err)
	}

	// Get losses count
	err = db.QueryRow(`
		SELECT COUNT(*) FROM bets b
		JOIN markets m ON b.market_id = m.id
		WHERE b.user_id = ? AND m.status = 'FINALIZED' AND m.outcome != b.outcome AND m.outcome != ''
	`, userID).Scan(&stats.Losses)
	if err != nil {
		return nil, fmt.Errorf("failed to get losses count: %w", err)
	}

	// Get total wagered
	err = db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM bets WHERE user_id = ?
	`, userID).Scan(&stats.TotalWager)
	if err != nil {
		return nil, fmt.Errorf("failed to get total wager: %w", err)
	}

	// Get total winnings (net profit)
	err = db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM transactions
		WHERE user_id = ? AND source_type = 'WIN_PAYOUT'
	`, userID).Scan(&stats.TotalWins)
	if err != nil {
		return nil, fmt.Errorf("failed to get total wins: %w", err)
	}

	// Calculate win rate
	if stats.TotalBets > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalBets) * 100
	}

	return stats, nil
}

// GetTopUsers returns the top users by balance for the leaderboard
func GetTopUsers(limit int) ([]LeaderboardEntry, error) {
	// Use ROW_NUMBER() for proper ranking
	rows, err := db.Query(`
		SELECT 
			ROW_NUMBER() OVER (ORDER BY balance DESC) as rank,
			username,
			first_name,
			balance
		FROM users
		ORDER BY balance DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	for rows.Next() {
		var entry LeaderboardEntry
		var username sql.NullString
		err := rows.Scan(&entry.Rank, &username, &entry.Name, &entry.Balance)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}

		if username.Valid {
			entry.Username = username.String
		} else {
			entry.Username = ""
		}
		entry.BalanceDisplay = fmt.Sprintf("%d", entry.Balance)

		leaderboard = append(leaderboard, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating leaderboard: %w", err)
	}

	return leaderboard, nil
}

// GetLastBailout returns the timestamp of the last bailout transaction for a user
// Returns (time.Time{}, false) if no bailout exists
func GetLastBailout(userID int64) (time.Time, bool, error) {
	var lastBailout time.Time
	err := db.QueryRow(`
		SELECT created_at FROM transactions
		WHERE user_id = ? AND source_type = 'BAILOUT'
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&lastBailout)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("failed to get last bailout: %w", err)
	}
	return lastBailout, true, nil
}

// ExecuteBailout executes a bailout transaction for a bankrupt user
// Sets balance to BailoutAmount (50000 cents = 500 WSC)
// Returns the new balance or an error
func ExecuteBailout(userID int64) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check current balance
	var currentBalance int64
	err = tx.QueryRow(`SELECT balance FROM users WHERE id = ?`, userID).Scan(&currentBalance)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("user not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	// Check if user is eligible (balance < threshold)
	if currentBalance >= BailoutBalanceThreshold {
		return 0, fmt.Errorf("balance_too_high: user has sufficient funds")
	}

	// Check cooldown
	lastBailout, hasBailout, err := GetLastBailout(userID)
	if err != nil {
		return 0, fmt.Errorf("failed to check bailout eligibility: %w", err)
	}
	if hasBailout && time.Since(lastBailout) < BailoutCooldown {
		return 0, fmt.Errorf("cooldown_active: last bailout was at %s", lastBailout.Format(time.RFC3339))
	}

	// Execute bailout: set balance to BailoutAmount
	// First get current balance, then update
	_, err = tx.Exec(`UPDATE users SET balance = ? WHERE id = ?`, BailoutAmount, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to update balance: %w", err)
	}

	// Log the bailout transaction
	_, err = tx.Exec(`
		INSERT INTO transactions (user_id, amount, source_type, description)
		VALUES (?, ?, 'BAILOUT', 'Emergency mortgage - free bailout')
	`, userID, BailoutAmount)
	if err != nil {
		return 0, fmt.Errorf("failed to log bailout transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit bailout: %w", err)
	}

	return BailoutAmount, nil
}
