package database

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/user/minicoinbase/backend/internal/models" // Import models package
)

// CreateUser inserts a new user into the database.
func CreateUser(ctx context.Context, username string, passwordHash string) (*models.User, error) {
	user := &models.User{
		Username: username,
		Password: passwordHash, // This is the hash
	}

	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2)
			  RETURNING id, created_at`

	err := DB.QueryRow(ctx, query, username, passwordHash).
		Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		// TODO: Check for specific errors like unique constraint violation
		return nil, err
	}

	return user, nil
}

// GetUserByUsername retrieves a user by their username.
func GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`

	err := DB.QueryRow(ctx, query, username).
		Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // User not found, return nil without error
		}
		return nil, err // Other database error
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `SELECT id, username, password_hash, created_at FROM users WHERE id = $1`

	err := DB.QueryRow(ctx, query, userID).
		Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}
