package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

const (
	createTableStmt = `
		CREATE TABLE IF NOT EXISTS apikeys (
			id TEXT PRIMARY KEY CHECK(length(id) = 36),
			auth_token TEXT CHECK(length(auth_token) = 64) NOT NULL,
			description VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT DATETIME
		);`
)

// GetConnection returns the singleton database connection pool
func GetConnection() (*sql.DB, error) {
	var err error
	once.Do(func() {
		db, err = sql.Open("sqlite3", "apikeys.sqlite")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

// PrepareDB ensures the necessary database tables are created
func PrepareDB(db *sql.DB) error {
	// Using context with timeout for better control
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Execute table creation with context
	_, err := db.ExecContext(ctx, createTableStmt)
	if err != nil {
		panic(fmt.Errorf("failed to create table: %w", err))
	}

	return nil
}
