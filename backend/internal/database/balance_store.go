package database

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/user/minicoinbase/backend/internal/models"
)

// GetBalance retrieves a user's balance for a specific asset.
// Returns nil, nil if the balance record doesn't exist.
func GetBalance(ctx context.Context, userID uuid.UUID, asset string) (*models.Balance, error) {
	balance := &models.Balance{}
	query := `SELECT user_id, asset, available, locked, updated_at
			  FROM balances WHERE user_id = $1 AND asset = $2`

	err := DB.QueryRow(ctx, query, userID, asset).
		Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Locked, &balance.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No balance record found for this asset yet
		}
		return nil, fmt.Errorf("error getting balance for user %s asset %s: %w", userID, asset, err)
	}
	return balance, nil
}

// GetOrCreateBalance retrieves a balance or creates it with zero values if it doesn't exist.
func GetOrCreateBalance(ctx context.Context, userID uuid.UUID, asset string) (*models.Balance, error) {
	balance, err := GetBalance(ctx, userID, asset)
	if err != nil {
		return nil, err // Database error
	}
	if balance != nil {
		return balance, nil // Balance exists
	}

	// Balance doesn't exist, create it
	newBalance := &models.Balance{
		UserID:    userID,
		Asset:     asset,
		Available: 0,
		Locked:    0,
	}
	query := `INSERT INTO balances (user_id, asset, available, locked)
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (user_id, asset) DO NOTHING -- Avoid race condition if created between check and insert
			  RETURNING updated_at` // Get the timestamp set by default NOW()

	err = DB.QueryRow(ctx, query, userID, asset, 0, 0).Scan(&newBalance.UpdatedAt)

	if err != nil {
		// If ErrNoRows, it means the ON CONFLICT clause was hit (or another Scan error occurred)
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict occurred, the row likely exists now, so re-fetch it.
			log.Printf("Conflict creating balance for user %s asset %s, re-fetching...", userID, asset)
			return GetBalance(ctx, userID, asset) // Return both balance and error from GetBalance
		}
		// Some other database or scan error occurred
		return nil, fmt.Errorf("error creating/scanning initial balance for user %s asset %s: %w", userID, asset, err)
	}

	// No error means the INSERT was successful
	return newBalance, nil // Successfully inserted
}

// GetUserBalances retrieves all balances for a given user.
func GetUserBalances(ctx context.Context, userID uuid.UUID) ([]*models.Balance, error) {
	balances := make([]*models.Balance, 0)
	query := `SELECT user_id, asset, available, locked, updated_at
			  FROM balances WHERE user_id = $1 ORDER BY asset`

	rows, err := DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying balances for user %s: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		balance := &models.Balance{}
		err := rows.Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Locked, &balance.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning balance row for user %s: %w", userID, err)
		}
		balances = append(balances, balance)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating balance rows for user %s: %w", userID, rows.Err())
	}

	return balances, nil
}

// LockFunds decreases available balance and increases locked balance for an asset.
// Requires an active transaction (tx) and checks for sufficient available funds.
func LockFunds(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount float64) error {
	// Ensure amount is positive
	if amount <= 0 {
		return fmt.Errorf("lock amount must be positive")
	}

	query := `UPDATE balances
			  SET available = available - $1, locked = locked + $1
			  WHERE user_id = $2 AND asset = $3 AND available >= $1`

	cmdTag, err := tx.Exec(ctx, query, amount, userID, asset)
	if err != nil {
		return fmt.Errorf("error locking funds for user %s asset %s: %w", userID, asset, err)
	}

	// Check if exactly one row was affected. If not, funds were insufficient or balance didn't exist.
	if cmdTag.RowsAffected() != 1 {
		// Attempt to get current balance for better error message
		// Note: This query runs within the SAME transaction tx
		currBalance, getErr := GetBalanceInTx(ctx, tx, userID, asset)
		if getErr != nil {
			return fmt.Errorf("insufficient funds for user %s asset %s (balance check failed: %w)", userID, asset, getErr)
		}
		if currBalance == nil {
			return fmt.Errorf("insufficient funds for user %s asset %s (balance not found)", userID, asset)
		}
		return fmt.Errorf("insufficient funds for user %s asset %s (available: %f, required: %f)",
			userID, asset, currBalance.Available, amount)
	}

	return nil
}

// UnlockFunds increases available balance and decreases locked balance.
// Typically used when an order is cancelled or partially filled.
// Requires an active transaction (tx).
func UnlockFunds(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount float64) error {
	// Ensure amount is positive
	if amount <= 0 {
		return fmt.Errorf("unlock amount must be positive")
	}

	query := `UPDATE balances
			  SET available = available + $1, locked = locked - $1
			  WHERE user_id = $2 AND asset = $3 AND locked >= $1`

	cmdTag, err := tx.Exec(ctx, query, amount, userID, asset)
	if err != nil {
		return fmt.Errorf("error unlocking funds for user %s asset %s: %w", userID, asset, err)
	}

	// Check if exactly one row was affected. If not, locked funds were insufficient or balance didn't exist.
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("failed to unlock sufficient locked funds for user %s asset %s (requested: %f)",
			userID, asset, amount)
	}

	return nil
}

// UpdateBalances adjusts available/locked funds after an order fill.
// Requires an active transaction (tx).
// For a buy fill: decrease quote locked, increase base available.
// For a sell fill: decrease base locked, increase quote available.
func UpdateBalancesForFill(ctx context.Context, tx pgx.Tx, userID uuid.UUID, baseAsset, quoteAsset string, baseAmount, quoteAmount float64, side string) error {
	var err error
	if side == "buy" {
		// Decrease locked quote asset (amount spent)
		query1 := `UPDATE balances SET locked = locked - $1 WHERE user_id = $2 AND asset = $3 AND locked >= $1`
		cmdTag1, err1 := tx.Exec(ctx, query1, quoteAmount, userID, quoteAsset)
		if err1 != nil {
			return fmt.Errorf("buy fill: failed to decrease locked %s: %w", quoteAsset, err1)
		}
		if cmdTag1.RowsAffected() != 1 {
			return fmt.Errorf("buy fill: failed to decrease sufficient locked %s", quoteAsset)
		}

		// Increase available base asset (amount bought)
		query2 := `INSERT INTO balances (user_id, asset, available, locked) VALUES ($1, $2, $3, 0)
				   ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + $3`
		_, err = tx.Exec(ctx, query2, userID, baseAsset, baseAmount)
		if err != nil {
			return fmt.Errorf("buy fill: failed to increase available %s: %w", baseAsset, err)
		}

	} else if side == "sell" {
		// Decrease locked base asset (amount sold)
		query1 := `UPDATE balances SET locked = locked - $1 WHERE user_id = $2 AND asset = $3 AND locked >= $1`
		cmdTag1, err1 := tx.Exec(ctx, query1, baseAmount, userID, baseAsset)
		if err1 != nil {
			return fmt.Errorf("sell fill: failed to decrease locked %s: %w", baseAsset, err1)
		}
		if cmdTag1.RowsAffected() != 1 {
			return fmt.Errorf("sell fill: failed to decrease sufficient locked %s", baseAsset)
		}

		// Increase available quote asset (amount received)
		query2 := `INSERT INTO balances (user_id, asset, available, locked) VALUES ($1, $2, $3, 0)
				   ON CONFLICT (user_id, asset) DO UPDATE SET available = balances.available + $3`
		_, err = tx.Exec(ctx, query2, userID, quoteAsset, quoteAmount)
		if err != nil {
			return fmt.Errorf("sell fill: failed to increase available %s: %w", quoteAsset, err)
		}
	} else {
		return fmt.Errorf("invalid side for fill update: %s", side)
	}
	return nil
}

// GetBalanceInTx retrieves a balance within a specific transaction.
func GetBalanceInTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string) (*models.Balance, error) {
	balance := &models.Balance{}
	query := `SELECT user_id, asset, available, locked, updated_at
			  FROM balances WHERE user_id = $1 AND asset = $2 FOR UPDATE` // Lock row within transaction

	err := tx.QueryRow(ctx, query, userID, asset).
		Scan(&balance.UserID, &balance.Asset, &balance.Available, &balance.Locked, &balance.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("tx error getting balance for user %s asset %s: %w", userID, asset, err)
	}
	return balance, nil
}

// GetOrCreateBalanceInTx retrieves or creates a balance within a transaction.
func GetOrCreateBalanceInTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string) (*models.Balance, error) {
	// Use the transaction-specific getter first
	balance, err := GetBalanceInTx(ctx, tx, userID, asset)
	if err != nil {
		return nil, err // Database error
	}
	if balance != nil {
		return balance, nil // Balance exists
	}

	// Balance doesn't exist, create it within the transaction
	newBalance := &models.Balance{
		UserID:    userID,
		Asset:     asset,
		Available: 0,
		Locked:    0,
	}
	query := `INSERT INTO balances (user_id, asset, available, locked)
			  VALUES ($1, $2, $3, $4)
			  ON CONFLICT (user_id, asset) DO NOTHING
			  RETURNING updated_at`

	// Use tx.QueryRow here
	err = tx.QueryRow(ctx, query, userID, asset, 0, 0).Scan(&newBalance.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Conflict occurred, re-fetch using the Tx version
			log.Printf("Tx Conflict creating balance for user %s asset %s, re-fetching...", userID, asset)
			return GetBalanceInTx(ctx, tx, userID, asset)
		}
		return nil, fmt.Errorf("tx error creating/scanning initial balance for user %s asset %s: %w", userID, asset, err)
	}
	return newBalance, nil
}

// TODO: Implement functions for updating balances (e.g., LockFunds, UnlockFunds, AddFunds, SubtractFunds)
// These will likely require transactions (pgx.Tx) to ensure atomicity, especially when placing/filling orders.
// Example structure (needs transaction handling):
/*
func LockFunds(ctx context.Context, tx pgx.Tx, userID uuid.UUID, asset string, amount float64) error {
	query := `UPDATE balances
			  SET available = available - $1, locked = locked + $1
			  WHERE user_id = $2 AND asset = $3 AND available >= $1`
	cmdTag, err := tx.Exec(ctx, query, amount, userID, asset)
	if err != nil {
		return fmt.Errorf("error locking funds: %w", err)
	}
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("insufficient available balance or balance not found")
	}
	return nil
}
*/
