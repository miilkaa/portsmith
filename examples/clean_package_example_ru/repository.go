package example

// repository.go — слой доступа к данным.
//
// Правила этого слоя:
//  1. Реализует интерфейс UserRepository из ports.go.
//  2. Знает про gorm, SQL, схему БД.
//  3. НЕ содержит бизнес-логики — только CRUD и запросы.
//  4. Преобразует ошибки хранилища в доменные ошибки (ErrUserNotFound и т.д.).
//     Это критично: сервис не должен видеть gorm.ErrRecordNotFound.
//
// Как тестировать:
//   - Интеграционные тесты с testkit.NewTestDB (SQLite in-memory).
//   - Тесты в repository_test.go.

import (
	"context"
	"errors"

	"github.com/miilkaa/portsmith/pkg/database"
	"github.com/miilkaa/portsmith/pkg/pagination"
	"gorm.io/gorm"
)

// Repository реализует UserRepository поверх GORM.
type Repository struct {
	// base — генерик-репозиторий для стандартных CRUD-операций.
	// Убирает бойлерплейт Create/FindByID/Update/Delete.
	base database.Repository[User]

	// db — прямой доступ к gorm.DB для специфичных запросов (FindByEmail, List).
	db *gorm.DB
}

// NewRepository создаёт новый Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		base: database.NewRepository[User](db),
		db:   db,
	}
}

// Create сохраняет нового пользователя в базе.
func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.base.Create(ctx, user)
}

// FindByID возвращает пользователя по первичному ключу.
// Возвращает ErrUserNotFound если запись не найдена — не gorm.ErrRecordNotFound.
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

// FindByEmail ищет пользователя по email.
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

// Update сохраняет изменения существующего пользователя.
func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.base.Update(ctx, user)
}

// Delete удаляет пользователя по ID (soft delete если у модели есть DeletedAt).
func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.base.Delete(ctx, id)
}

// List возвращает страницу пользователей с опциональной фильтрацией.
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
