// Package database provides GORM utilities for Clean Architecture backends:
//
//   - Connect: open a database connection (Postgres or SQLite)
//   - Register: AutoMigrate domain models at startup
//   - Repository[T]: generic CRUD base repository
//   - WithTx: run a function inside a transaction with automatic commit/rollback
//
// Usage:
//
//	db, err := database.Connect(database.Config{DSN: os.Getenv("DATABASE_URL")})
//	if err != nil { ... }
//
//	database.Register(db, &user.User{}, &order.Order{})
//
//	repo := database.NewRepository[user.User](db.DB())
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/miilkaa/portsmith/pkg/apperrors"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Driver identifies the database engine.
type Driver string

const (
	DriverPostgres Driver = "postgres"
	DriverSQLite   Driver = "sqlite"
)

// Config holds database connection settings.
type Config struct {
	// Driver selects the database engine. Defaults to Postgres.
	Driver Driver

	// DSN is the data source name.
	// Postgres: "host=... user=... dbname=... sslmode=disable"
	// SQLite:   ":memory:" or "/path/to/file.db"
	DSN string

	// Silent disables GORM query logging (useful in tests).
	Silent bool
}

// DB wraps *gorm.DB to keep the portsmith API decoupled from gorm import paths.
type DB struct {
	g *gorm.DB
}

// DB returns the underlying *gorm.DB for queries that need it directly.
func (d *DB) DB() *gorm.DB {
	return d.g
}

// Connect opens a database connection based on Config.
func Connect(cfg Config) (*DB, error) {
	logLevel := gormlogger.Info
	if cfg.Silent {
		logLevel = gormlogger.Silent
	}

	gcfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	}

	driver := cfg.Driver
	if driver == "" {
		driver = DriverPostgres
	}

	var dialector gorm.Dialector
	switch driver {
	case DriverPostgres:
		dialector = postgres.Open(cfg.DSN)
	case DriverSQLite:
		dialector = sqlite.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("database: unsupported driver %q", driver)
	}

	g, err := gorm.Open(dialector, gcfg)
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}
	return &DB{g: g}, nil
}

// Register runs AutoMigrate for all provided models.
// Call once at application startup after Connect.
//
//	database.Register(db, &user.User{}, &order.Order{})
func Register(db *DB, models ...any) error {
	if err := db.g.AutoMigrate(models...); err != nil {
		return fmt.Errorf("database: automigrate: %w", err)
	}
	return nil
}

// WithTx runs fn inside a transaction.
// Commits on success, rolls back on any error returned by fn.
func WithTx(ctx context.Context, db *DB, fn func(tx *DB) error) error {
	return db.g.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&DB{g: tx})
	})
}

// Repository is a generic base repository for standard CRUD operations.
// Embed or use directly to avoid per-entity boilerplate.
//
// FindByID returns apperrors.NotFound when no record exists,
// keeping gorm internals from leaking into the service layer.
type Repository[T any] struct {
	db *gorm.DB
}

// NewRepository creates a new generic Repository backed by the given *gorm.DB.
func NewRepository[T any](db *gorm.DB) Repository[T] {
	return Repository[T]{db: db}
}

// Create inserts a new record and populates the primary key on the entity.
func (r Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// FindByID retrieves a record by primary key.
// Returns apperrors.NotFound (code NOT_FOUND) when no record is found.
func (r Repository[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.NotFound("record not found")
		}
		return nil, err
	}
	return &entity, nil
}

// Update saves all fields of an existing record.
func (r Repository[T]) Update(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete removes a record by primary key.
func (r Repository[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	return r.db.WithContext(ctx).Delete(&entity, id).Error
}
