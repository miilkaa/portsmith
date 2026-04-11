package example

// ports.go — интерфейсы-порты пакета.
//
// В реальном проекте этот файл генерируется командой:
//
//	portsmith gen ./internal/example
//
// Порты определяются в том пакете, который их ИСПОЛЬЗУЕТ, а не в том,
// который их реализует. Это — инверсия зависимостей (Dependency Inversion Principle).
//
// Handler использует UserService → UserService определён здесь.
// Service использует UserRepository → UserRepository определён здесь.
//
// Важно: интерфейсы содержат ТОЛЬКО те методы, которые реально вызываются.
// Не нужно копировать весь API Repository в интерфейс — только используемое.
// Это принцип Interface Segregation и минимальных интерфейсов в Go.

import (
	"context"

	"github.com/miilkaa/portsmith/pkg/pagination"
)

// UserRepository — порт хранилища, используемый сервисом.
//
// Service зависит от этого интерфейса, а не от конкретного *Repository.
// Это позволяет тестировать сервис без базы данных — достаточно мок-репозитория.
//
// Compile-time проверка: var _ UserRepository = (*Repository)(nil)
// Если Repository не реализует интерфейс — ошибка компиляции, а не runtime.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uint) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter ListFilter, page pagination.OffsetPage) ([]*User, int64, error)
}

// UserService — порт бизнес-логики, используемый хендлером.
//
// Handler зависит от этого интерфейса, а не от конкретного *Service.
// Позволяет тестировать хендлер без сервиса и без базы данных.
type UserService interface {
	Create(ctx context.Context, params CreateParams) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
	Update(ctx context.Context, id uint, params UpdateParams, callerID uint) (*User, error)
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filter ListFilter, page pagination.OffsetPage) ([]*User, int64, error)
}

// Compile-time assertions — гарантируют соответствие реализаций портам.
// Ошибка компиляции "does not implement" сразу укажет на несоответствие.
var (
	_ UserRepository = (*Repository)(nil)
	_ UserService    = (*Service)(nil)
)
