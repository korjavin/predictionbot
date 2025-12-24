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

// MarketStatus represents the status of a market
type MarketStatus string

const (
	MarketStatusActive     MarketStatus = "ACTIVE"
	MarketStatusLocked     MarketStatus = "LOCKED"
	MarketStatusResolved   MarketStatus = "RESOLVED"
	MarketStatusDisputed   MarketStatus = "DISPUTED"
	MarketStatusFinalized  MarketStatus = "FINALIZED"
)

// Market represents a prediction market
type Market struct {
	ID         int64        `json:"id" db:"id"`
	CreatorID  int64        `json:"creator_id" db:"creator_id"`
	Question   string       `json:"question" db:"question"`
	ImageURL   string       `json:"image_url,omitempty" db:"image_url"`
	Status     MarketStatus `json:"status" db:"status"`
	Outcome    string       `json:"outcome,omitempty" db:"outcome"`
	ResolvedAt time.Time    `json:"resolved_at,omitempty" db:"resolved_at"`
	ExpiresAt  time.Time    `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time    `json:"created_at" db:"created_at"`
}

// MarketResponse is the API response for a market
type MarketResponse struct {
	ID          int64  `json:"id"`
	Question    string `json:"question"`
	CreatorName string `json:"creator_name"`
	ExpiresAt   string `json:"expires_at"`
	PoolYes     int64  `json:"pool_yes"`
	PoolNo      int64  `json:"pool_no"`
}

// Outcome represents a betting outcome
type Outcome string

const (
	OutcomeYes Outcome = "YES"
	OutcomeNo  Outcome = "NO"
)

// Bet represents a bet placed on a market
type Bet struct {
	ID       int64     `json:"id" db:"id"`
	UserID   int64     `json:"user_id" db:"user_id"`
	MarketID int64     `json:"market_id" db:"market_id"`
	Outcome  Outcome   `json:"outcome" db:"outcome"`
	Amount   int64     `json:"amount" db:"amount"` // in cents
	PlacedAt time.Time `json:"placed_at" db:"placed_at"`
}
