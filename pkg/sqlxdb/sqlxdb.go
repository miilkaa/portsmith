// Package sqlxdb provides sqlx + PostgreSQL (pgx stdlib driver) helpers:
//
//   - Connect with pool settings
//   - WithTx for transactional work
//
// Migrations are not handled here; use goose, atlas, or your tool of choice.
//
// Usage:
//
//	db, err := sqlxdb.Connect(sqlxdb.Config{URL: os.Getenv("DATABASE_URL")})
//	if err != nil { ... }
//	defer db.Close()
package sqlxdb

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Config holds database connection settings for PostgreSQL via pgx stdlib driver.
type Config struct {
	URL string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Connect opens a *sqlx.DB using driver name "pgx".
func Connect(cfg Config) (*sqlx.DB, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("sqlxdb: URL is required")
	}
	db, err := sqlx.Connect("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("sqlxdb: connect: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	return db, nil
}

// WithTx runs fn inside a transaction. Commits on nil error, rolls back otherwise.
func WithTx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
