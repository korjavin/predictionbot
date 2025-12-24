package storage

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID         int64     `json:"id" db:"id"`
	TelegramID int64     `json:"telegram_id" db:"telegram_id"`
	Username   string    `json:"username" db:"username"`
	FirstName  string    `json:"first_name" db:"first_name"`
	Balance    int64     `json:"balance" db:"balance"` // in cents (1000 = 10.00)
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// Transaction represents a balance change
type Transaction struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Amount      int64     `json:"amount" db:"amount"`           // can be negative
	SourceType  string    `json:"source_type" db:"source_type"` // 'WELCOME_BONUS', 'BET', 'WIN'
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
