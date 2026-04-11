// Package database provides GORM utilities for portsmith-based applications.
//
// # Connecting
//
//	db, err := database.Connect(database.Config{
//	    DSN: os.Getenv("DATABASE_URL"),
//	})
//
// For SQLite (tests and local development):
//
//	db, err := database.Connect(database.Config{
//	    Driver: database.DriverSQLite,
//	    DSN:    ":memory:",
//	    Silent: true,
//	})
//
// # Auto-migration
//
// Register domain models at application startup:
//
//	database.Register(db, &user.User{}, &order.Order{})
//
// This calls gorm.AutoMigrate under the hood. It adds columns and indexes
// but never removes them (safe migrations).
//
// # Generic repository
//
// Use database.Repository[T] to avoid CRUD boilerplate:
//
//	type Repository struct {
//	    base database.Repository[User]
//	    db   *gorm.DB
//	}
//
//	func NewRepository(db *gorm.DB) *Repository {
//	    return &Repository{base: database.NewRepository[User](db), db: db}
//	}
//
//	func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
//	    // custom query — only this needs to be written manually
//	}
//
// # Transactions
//
//	err := database.WithTx(ctx, db, func(tx *database.DB) error {
//	    txRepo := NewRepository(tx.DB())
//	    return txRepo.Create(ctx, entity)
//	})
//
// The transaction is committed on nil return, rolled back on any error.
package database
