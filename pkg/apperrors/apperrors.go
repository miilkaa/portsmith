// Package apperrors provides typed domain errors with HTTP status mapping.
//
// Domain errors are defined in the domain layer without any HTTP knowledge.
// The server middleware maps AppError types to HTTP status codes automatically.
//
// Usage in domain layer:
//
//	var ErrUserNotFound = apperrors.NotFound("user not found")
//	var ErrEmailTaken   = apperrors.Conflict("email already taken")
//
// Usage in server middleware:
//
//	status := apperrors.HTTPStatus(err)  // 404, 409, etc.
package apperrors

import (
	"fmt"
	"net/http"
)

// Code represents the type of application error.
type Code string

const (
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeBadRequest   Code = "BAD_REQUEST"
	CodeForbidden    Code = "FORBIDDEN"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeInternal     Code = "INTERNAL"
)

// AppError is a typed domain error with an associated code and optional details.
// Use errors.Is for sentinel error comparison.
type AppError struct {
	code    Code
	message string
	details map[string]any
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.message
}

// Code returns the error classification code.
func (e *AppError) Code() Code {
	return e.code
}

// Details returns optional structured error details (e.g. validation field errors).
func (e *AppError) Details() map[string]any {
	return e.details
}

// Is enables errors.Is comparison for sentinel errors.
// Two AppErrors are equal when both their code and message match.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.code == t.code && e.message == t.message
}

func newError(code Code, msg string) *AppError {
	return &AppError{code: code, message: msg}
}

// NotFound creates a NOT_FOUND domain error (maps to HTTP 404).
func NotFound(msg string) *AppError { return newError(CodeNotFound, msg) }

// Conflict creates a CONFLICT domain error (maps to HTTP 409).
func Conflict(msg string) *AppError { return newError(CodeConflict, msg) }

// BadRequest creates a BAD_REQUEST domain error (maps to HTTP 400).
func BadRequest(msg string) *AppError { return newError(CodeBadRequest, msg) }

// Forbidden creates a FORBIDDEN domain error (maps to HTTP 403).
func Forbidden(msg string) *AppError { return newError(CodeForbidden, msg) }

// Unauthorized creates an UNAUTHORIZED domain error (maps to HTTP 401).
func Unauthorized(msg string) *AppError { return newError(CodeUnauthorized, msg) }

// Internal creates an INTERNAL domain error (maps to HTTP 500).
func Internal(msg string, args ...any) *AppError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return newError(CodeInternal, msg)
}

// WithDetails returns a copy of the AppError enriched with structured details.
// The resulting error still matches the original via errors.Is.
func WithDetails(err *AppError, details map[string]any) *AppError {
	return &AppError{
		code:    err.code,
		message: err.message,
		details: details,
	}
}

// HTTPStatus maps an AppError code to the corresponding HTTP status code.
// Returns 500 for any unrecognised code or non-AppError errors.
func HTTPStatus(err error) int {
	var appErr *AppError
	if !IsAppError(err) {
		return http.StatusInternalServerError
	}
	// Direct type assertion after IsAppError check.
	appErr, _ = err.(*AppError)
	switch appErr.code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeForbidden:
		return http.StatusForbidden
	case CodeUnauthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError reports whether err is an *AppError.
func IsAppError(err error) bool {
	var appErr *AppError
	return As(err, &appErr)
}

// IsCode reports whether err is an *AppError with the given code.
func IsCode(err error, code Code) bool {
	var appErr *AppError
	if !As(err, &appErr) {
		return false
	}
	return appErr.code == code
}

// As is a convenience wrapper around errors.As for *AppError.
func As(err error, target **AppError) bool {
	if err == nil {
		return false
	}
	// Direct type assertion first (fast path for non-wrapped errors).
	if ae, ok := err.(*AppError); ok {
		*target = ae
		return true
	}
	// Walk the error chain for wrapped AppErrors.
	type unwrapper interface{ Unwrap() error }
	for {
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == nil {
			return false
		}
		if ae, ok := err.(*AppError); ok {
			*target = ae
			return true
		}
	}
}
