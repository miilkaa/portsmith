// Package database provides GORM utilities: connection, auto-migration registry,
// a generic Repository[T] for standard CRUD, and transaction helpers.
package database

import (
	"context"

	"gorm.io/gorm"
)

// Repository is a generic base repository for standard CRUD operations.
// Embed it in your domain repository to avoid repeating boilerplate.
type Repository[T any] struct {
	db *gorm.DB
}

// NewRepository creates a new generic Repository.
func NewRepository[T any](db *gorm.DB) Repository[T] {
	return Repository[T]{db: db}
}

// Create inserts a new record.
func (r Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// FindByID retrieves a record by primary key.
func (r Repository[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).First(&entity, id).Error
	if err != nil {
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
