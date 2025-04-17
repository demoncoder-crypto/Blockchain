package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/user/minicoinbase/backend/internal/models"
)

// CreateOrder inserts a new order into the database.
// Note: This function assumes balance checks and locking have happened *before* calling it,
// ideally within a transaction.
func CreateOrder(ctx context.Context, tx pgx.Tx, order *models.Order) error {
	query := `INSERT INTO orders (user_id, symbol, type, side, price, quantity, status)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)
			  RETURNING id, created_at, updated_at`

	// Use the transaction (tx) if provided, otherwise use the pool (DB)
	querier := Querier(tx)

	err := querier.QueryRow(ctx, query,
		order.UserID, order.Symbol, order.Type, order.Side,
		order.Price, // Note: Handle NULL for market orders if necessary in model/handler
		order.Quantity, order.Status,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("error creating order for user %s: %w", order.UserID, err)
	}
	return nil
}

// GetUserOrders retrieves all non-cancelled orders for a specific user.
func GetUserOrders(ctx context.Context, userID uuid.UUID) ([]*models.Order, error) {
	orders := make([]*models.Order, 0)
	// Exclude cancelled orders, sort by creation time descending
	query := `SELECT id, user_id, symbol, type, side, price, quantity, status, created_at, updated_at
			  FROM orders
			  WHERE user_id = $1 AND status != 'cancelled'
			  ORDER BY created_at DESC`

	rows, err := DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying orders for user %s: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		order := &models.Order{}
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Symbol, &order.Type, &order.Side,
			&order.Price, &order.Quantity, &order.Status, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning order row for user %s: %w", userID, err)
		}
		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating order rows for user %s: %w", userID, rows.Err())
	}

	return orders, nil
}

// GetOrderByID retrieves a specific order by its ID.
func GetOrderByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	order := &models.Order{}
	query := `SELECT id, user_id, symbol, type, side, price, quantity, status, created_at, updated_at
			  FROM orders WHERE id = $1`

	err := DB.QueryRow(ctx, query, orderID).Scan(
		&order.ID, &order.UserID, &order.Symbol, &order.Type, &order.Side,
		&order.Price, &order.Quantity, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Order not found
		}
		return nil, fmt.Errorf("error getting order by id %s: %w", orderID, err)
	}
	return order, nil
}

// CancelOrder updates an order's status to 'cancelled' within a transaction.
// It returns the details of the order *before* cancellation (for fund unlocking).
// It checks if the order belongs to the user and is currently cancellable (e.g., 'open').
func CancelOrder(ctx context.Context, tx pgx.Tx, userID uuid.UUID, orderID uuid.UUID) (*models.Order, error) {
	// 1. Get the order details first, ensuring it belongs to the user and is in a cancellable state.
	//    Use FOR UPDATE to lock the row within the transaction.
	order := &models.Order{}
	get_query := `SELECT id, user_id, symbol, type, side, price, quantity, status
				   FROM orders
				   WHERE id = $1 AND user_id = $2 FOR UPDATE`

	err := tx.QueryRow(ctx, get_query, orderID, userID).Scan(
		&order.ID, &order.UserID, &order.Symbol, &order.Type, &order.Side,
		&order.Price, &order.Quantity, &order.Status,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Order not found OR doesn't belong to the user
			return nil, fmt.Errorf("order not found or permission denied")
		}
		return nil, fmt.Errorf("error retrieving order %s for cancellation: %w", orderID, err)
	}

	// 2. Check if the order is actually cancellable
	if order.Status != "open" { // Only open orders can be cancelled (adjust if partial fills allowed cancellation)
		return nil, fmt.Errorf("order %s is not in a cancellable state (status: %s)", orderID, order.Status)
	}

	// 3. Update the status to 'cancelled'
	update_query := `UPDATE orders SET status = 'cancelled', updated_at = NOW()
					 WHERE id = $1 AND status = 'open'` // Double check status

	cmdTag, err := tx.Exec(ctx, update_query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error updating order %s status to cancelled: %w", orderID, err)
	}

	if cmdTag.RowsAffected() != 1 {
		// This should ideally not happen due to the FOR UPDATE lock and status check,
		// but indicates a potential race condition or logic error if it does.
		return nil, fmt.Errorf("failed to update order %s status (concurrent modification?)", orderID)
	}

	// Return the order details *before* it was cancelled (status will be 'open' here)
	return order, nil
}

// Helper type to allow using either pgx.Pool or pgx.Tx
type PgxQuerier interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

// Querier returns the transaction if not nil, otherwise the pool.
func Querier(tx pgx.Tx) PgxQuerier {
	if tx != nil {
		return tx
	}
	return DB // DB is the global *pgxpool.Pool from postgres.go
}

// Need to import pgconn for CommandTag
// import "github.com/jackc/pgx/v5/pgconn"
// import "errors" // Already imported usually
