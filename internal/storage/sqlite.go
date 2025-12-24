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
	// WelcomeBonusAmount is the welcome bonus amount in cents (1000 WSC = 100000 cents)
	WelcomeBonusAmount int64 = 100000
)

var db *sql.DB

// InitDB initializes the SQLite database connection with WAL mode
func InitDB(dbPath string) error {
	var err error

	// Ensure the directory exists
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}

	db, err = sql.Open("sqlite", absPath)
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
	err := db.QueryRow(`
		SELECT id, creator_id, question, image_url, status, expires_at, created_at
		FROM markets
		WHERE id = ?
	`, id).Scan(
		&market.ID,
		&market.CreatorID,
		&market.Question,
		&market.ImageURL,
		&market.Status,
		&market.ExpiresAt,
		&market.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get market by id: %w", err)
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
		err := rows.Scan(
			&market.ID,
			&market.CreatorID,
			&market.Question,
			&market.ImageURL,
			&market.Status,
			&market.ExpiresAt,
			&market.CreatedAt,
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
