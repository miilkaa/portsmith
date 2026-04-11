// Package example демонстрирует полную структуру пакета в Clean Architecture.
//
// Этот пакет — эталонный пример для команд, использующих portsmith.
// Каждый файл отвечает за строго один слой. Зависимости направлены только внутрь:
//
//	Handler → ServicePort ← Service → RepositoryPort ← Repository
//
// Файлы в порядке от ядра наружу:
//
//	domain.go      — доменные типы (ядро, ноль зависимостей)
//	errors.go      — доменные ошибки
//	ports.go       — интерфейсы-порты (обычно генерируется portsmith gen)
//	service.go     — бизнес-логика
//	repository.go  — реализация хранилища
//	handler.go     — HTTP-адаптер
//	dto.go         — структуры запросов и ответов
//	mappers.go     — преобразование domain ↔ DTO
package example

import "time"

// User — центральная доменная сущность пакета.
//
// Правило: domain.go не импортирует ни database/sql, ни net/http, ни gorm.
// Это чистые Go-типы. Их можно безопасно передавать между слоями.
//
// GORM-теги (`gorm:"..."`) — компромисс ради AutoMigrate.
// В строго чистой архитектуре DB-модель была бы отдельным типом в repository.go,
// а здесь лежали бы только бизнес-поля. Для большинства проектов тегов на домене
// достаточно — разделяй при необходимости.
type User struct {
	ID        uint     `gorm:"primaryKey"`
	Email     string   `gorm:"uniqueIndex;not null"`
	Name      string   `gorm:"not null"`
	Role      UserRole `gorm:"default:'user'"`
	Active    bool     `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRole — перечисление ролей. Определяется в домене, не в БД.
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// CreateParams — входные данные для создания пользователя.
// Используется сервисом: handler преобразует DTO → CreateParams,
// сервис принимает CreateParams → вызывает репозиторий.
//
// Почему не передавать DTO прямо в сервис?
// Сервис не должен знать про HTTP-специфику (json-теги, validator-теги).
// CreateParams — это чистый "язык" бизнес-операции.
type CreateParams struct {
	Email string
	Name  string
	Role  UserRole
}

// UpdateParams — входные данные для частичного обновления.
// Указатели означают "поле передано" / "поле не передано" (partial update).
type UpdateParams struct {
	Name   *string
	Role   *UserRole
	Active *bool
}

// ListFilter — параметры фильтрации для запросов списка.
type ListFilter struct {
	Role   *UserRole
	Active *bool
}
