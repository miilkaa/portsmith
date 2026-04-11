package example

// repository.go — data access layer.
//
// Rules for this layer:
//  1. Implements the UserRepository interface from ports.go.
//  2. Knows about gorm, SQL, and the database schema.
//  3. Must NOT contain business logic — only CRUD and queries.
//  4. Translates storage errors into domain errors (ErrUserNotFound, etc.).
//     This is critical: the service must never see gorm.ErrRecordNotFound.
//
// How to test:
//   - Integration tests with testkit.NewTestDB (SQLite in-memory).
//   - Tests live in repository_test.go.

import (
	"context"
	"errors"

	"github.com/miilkaa/portsmith/pkg/database"
	"github.com/miilkaa/portsmith/pkg/pagination"
	"gorm.io/gorm"
)

// Repository implements UserRepository on top of GORM.
type Repository struct {
	// base is the generic repository that handles standard CRUD operations.
	// It eliminates boilerplate for Create / FindByID / Update / Delete.
	base database.Repository[User]

	// db provides direct access to *gorm.DB for specialised queries
	// such as FindByEmail and List with dynamic filters.
	db *gorm.DB
}

// NewRepository creates a new Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		base: database.NewRepository[User](db),
		db:   db,
	}
}

// Create persists a new user to the database.
func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.base.Create(ctx, user)
}

// FindByID returns a user by primary key.
// Returns ErrUserNotFound when the record does not exist — never gorm.ErrRecordNotFound.
func (r *Repository) FindByID(ctx context.Context, id uint) (*User, error) {
	user, err := r.base.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// FindByEmail looks up a user by email address.
func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Update saves changes to an existing user.
func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.base.Update(ctx, user)
}

// Delete removes a user by ID (soft delete if the model has a DeletedAt field).
func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

// List returns a page of users with optional filtering.
func (r *Repository) List(ctx context.Context, filter ListFilter, page pagination.OffsetPage) ([]*User, int64, error) {
	var users []*User
	var total int64

	query := r.db.WithContext(ctx).Model(&User{})

	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.Active != nil {
		query = query.Where("active = ?", *filter.Active)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Offset(page.Offset()).
		Limit(page.Limit()).
		Find(&users).Error

	return users, total, err
}
