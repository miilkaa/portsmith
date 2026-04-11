package apperrors_test

// apperrors_test.go — контрактные тесты для pkg/apperrors.
//
// Контракт определяет публичный API пакета:
//  1. Конструкторы создают ошибки с нужными кодами.
//  2. errors.Is работает для sentinel errors.
//  3. HTTPStatus возвращает корректный HTTP-статус для каждого типа.
//  4. WithDetails добавляет детали без изменения базовой ошибки.
//  5. Unwrap позволяет оборачивать ошибки и сохранять errors.Is.

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/miilkaa/portsmith/pkg/apperrors"
)

func TestConstructors_createCorrectCodes(t *testing.T) {
	cases := []struct {
		name string
		err  *apperrors.AppError
		want apperrors.Code
	}{
		{"NotFound", apperrors.NotFound("x"), apperrors.CodeNotFound},
		{"Conflict", apperrors.Conflict("x"), apperrors.CodeConflict},
		{"BadRequest", apperrors.BadRequest("x"), apperrors.CodeBadRequest},
		{"Forbidden", apperrors.Forbidden("x"), apperrors.CodeForbidden},
		{"Unauthorized", apperrors.Unauthorized("x"), apperrors.CodeUnauthorized},
		{"Internal", apperrors.Internal("x"), apperrors.CodeInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code() != tc.want {
				t.Errorf("expected code %s, got %s", tc.want, tc.err.Code())
			}
		})
	}
}

func TestError_implementsErrorInterface(t *testing.T) {
	var err error = apperrors.NotFound("user not found")
	if err.Error() != "user not found" {
		t.Errorf("unexpected message: %s", err.Error())
	}
}

func TestIs_sentinelErrors(t *testing.T) {
	// Это ключевой контракт: sentinel errors в domain layer сравниваются через errors.Is.
	var ErrUserNotFound = apperrors.NotFound("user not found")

	// Один и тот же sentinel.
	if !errors.Is(ErrUserNotFound, ErrUserNotFound) {
		t.Error("sentinel error must match itself via errors.Is")
	}

	// Разные ошибки не совпадают.
	ErrOther := apperrors.NotFound("other not found")
	if errors.Is(ErrUserNotFound, ErrOther) {
		t.Error("different sentinel errors must not match")
	}
}

func TestIs_wrappedError(t *testing.T) {
	sentinel := apperrors.NotFound("user not found")
	wrapped := fmt.Errorf("repository: %w", sentinel)

	if !errors.Is(wrapped, sentinel) {
		t.Error("wrapped sentinel error must be found via errors.Is")
	}
}

func TestHTTPStatus_mapsCorrectly(t *testing.T) {
	cases := []struct {
		err    *apperrors.AppError
		status int
	}{
		{apperrors.NotFound("x"), http.StatusNotFound},
		{apperrors.Conflict("x"), http.StatusConflict},
		{apperrors.BadRequest("x"), http.StatusBadRequest},
		{apperrors.Forbidden("x"), http.StatusForbidden},
		{apperrors.Unauthorized("x"), http.StatusUnauthorized},
		{apperrors.Internal("x"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(string(tc.err.Code()), func(t *testing.T) {
			got := apperrors.HTTPStatus(tc.err)
			if got != tc.status {
				t.Errorf("expected HTTP %d, got %d", tc.status, got)
			}
		})
	}
}

func TestWithDetails_addsDetails(t *testing.T) {
	base := apperrors.BadRequest("validation failed")
	enriched := apperrors.WithDetails(base, map[string]any{
		"field": "email",
		"issue": "invalid format",
	})

	if !errors.Is(enriched, base) {
		t.Error("enriched error must match base via errors.Is")
	}

	details := enriched.Details()
	if details["field"] != "email" {
		t.Errorf("expected field=email, got %v", details["field"])
	}
}

func TestIsAppError_detectsType(t *testing.T) {
	appErr := apperrors.NotFound("x")
	var stdErr error = errors.New("standard error")

	if !apperrors.IsAppError(appErr) {
		t.Error("AppError must be detected as app error")
	}
	if apperrors.IsAppError(stdErr) {
		t.Error("standard error must not be detected as app error")
	}
}

func TestIsCode_checksByCode(t *testing.T) {
	err := apperrors.NotFound("user not found")

	if !apperrors.IsCode(err, apperrors.CodeNotFound) {
		t.Error("must detect CodeNotFound")
	}
	if apperrors.IsCode(err, apperrors.CodeConflict) {
		t.Error("must not detect CodeConflict for NotFound error")
	}
}
