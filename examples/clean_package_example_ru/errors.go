package example

// errors.go — доменные ошибки пакета.
//
// Правило: ошибки не знают про HTTP-коды. Они описывают бизнес-ситуацию.
// Middleware в pkg/server автоматически маппит типы apperrors на HTTP-статусы:
//
//	apperrors.NotFound    → 404
//	apperrors.Conflict    → 409
//	apperrors.BadRequest  → 400
//	apperrors.Forbidden   → 403
//
// Почему переменные, а не типы?
// Сравнение через errors.Is работает с sentinel errors (переменными).
// Это стандартный Go-паттерн для доменных ошибок.

import "github.com/miilkaa/portsmith/pkg/apperrors"

var (
	// ErrUserNotFound возвращается когда пользователь не найден по ID или email.
	ErrUserNotFound = apperrors.NotFound("user not found")

	// ErrEmailTaken возвращается при попытке создать пользователя
	// с email, который уже занят.
	ErrEmailTaken = apperrors.Conflict("email already taken")

	// ErrCannotDeactivateSelf возвращается когда администратор
	// пытается деактивировать собственный аккаунт.
	ErrCannotDeactivateSelf = apperrors.BadRequest("cannot deactivate your own account")

	// ErrInsufficientPermissions возвращается при попытке выполнить
	// операцию, требующую прав администратора.
	ErrInsufficientPermissions = apperrors.Forbidden("insufficient permissions")
)
