package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account
type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // Store hash, exclude from JSON responses
	CreatedAt time.Time `json:"created_at"`
}

// Order represents a trading order
type Order struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Symbol    string    `json:"symbol"`          // e.g., "BTC-USD"
	Type      string    `json:"type"`            // e.g., "limit", "market"
	Side      string    `json:"side"`            // e.g., "buy", "sell"
	Price     float64   `json:"price,omitempty"` // Only for limit orders
	Quantity  float64   `json:"quantity"`
	Status    string    `json:"status"` // e.g., "open", "filled", "cancelled"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Balance represents a user's balance for a specific asset
type Balance struct {
	UserID    uuid.UUID `json:"user_id"`
	Asset     string    `json:"asset"` // e.g., "USD", "BTC"
	Available float64   `json:"available"`
	Locked    float64   `json:"locked"` // Funds locked in open orders
	UpdatedAt time.Time `json:"updated_at"`
}
