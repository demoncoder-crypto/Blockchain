package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// InitDB initializes the database connection pool.
func InitDB() {
	// DATABASE_URL="postgres://user:password@host:port/database"
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Provide a default for local development if not set
		dbURL = "postgres://postgres:password@localhost:5432/minicoinbase?sslmode=disable"
		log.Println("DATABASE_URL not set, using default:", dbURL)
	}

	var err error
	DB, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// Optional: Ping the database to verify connection
	err = DB.Ping(context.Background())
	if err != nil {
		log.Fatalf("Database ping failed: %v\n", err)
	}

	fmt.Println("Successfully connected to the database!")

	// TODO: Run database migrations here if needed
	// migrateDB(DB)
}

// CloseDB closes the database connection pool.
func CloseDB() {
	if DB != nil {
		DB.Close()
		fmt.Println("Database connection closed.")
	}
}

// TODO: Add migration logic (e.g., using migrate library)
/*
func migrateDB(pool *pgxpool.Pool) {
	// Example using golang-migrate (requires adding the dependency)
	// driver, err := postgres.WithInstance(pool.DB(), &postgres.Config{})
	// m, err := migrate.NewWithDatabaseInstance(
	// 	"file://./migrations", // Path to migration files
	// 	"postgres", driver)
	// if err != nil {
	// 	log.Fatalf("Migration setup failed: %v", err)
	// }
	// if err := m.Up(); err != nil && err != migrate.ErrNoChange {
	// 	log.Fatalf("Migration failed: %v", err)
	// }
	// log.Println("Database migrations applied successfully.")

    log.Println("Database migrations would run here...")
}
*/
