package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenAndMigrate(sqlitePath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir db dir: %w", err)
	}

	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite single writer; keep it predictable.

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			amount REAL NOT NULL,
			currency TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS payments (
			id TEXT PRIMARY KEY,
			order_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			provider_payment_id TEXT,
			status TEXT NOT NULL,
			redirect_url TEXT,
			raw_last_event TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(order_id) REFERENCES orders(id)
		);`,
		`CREATE TABLE IF NOT EXISTS webhook_events (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			event_key TEXT NOT NULL,
			received_at TEXT NOT NULL,
			processed_at TEXT,
			UNIQUE(provider, event_key)
		);`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

