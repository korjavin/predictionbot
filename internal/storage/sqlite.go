package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"

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

	// Create indexes for better query performance
	createIndexes := `
		CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
		CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
	`

	_, err := db.Exec(usersTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(transactionsTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createIndexes)
	if err != nil {
		return err
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
