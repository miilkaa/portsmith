// Package apperrors provides typed domain errors with HTTP status mapping.
// Errors are defined in domain layer without HTTP knowledge.
// The server middleware maps error types to HTTP status codes automatically.
package apperrors

import "fmt"

// Code represents an application error type.
type Code string

const (
	CodeNotFound   Code = "NOT_FOUND"
	CodeConflict   Code = "CONFLICT"
	CodeBadRequest Code = "BAD_REQUEST"
	CodeForbidden  Code = "FORBIDDEN"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeInternal   Code = "INTERNAL"
)

// AppError is a typed domain error.
type AppError struct {
	code    Code
	message string
	details map[string]any
}

func (e *AppError) Error() string {
	return e.message
}

// Code returns the error code.
func (e *AppError) Code() Code {
	return e.code
}

// Details returns optional error details.
func (e *AppError) Details() map[string]any {
	return e.details
}

// Is enables errors.Is comparison for sentinel errors.
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

// NotFound creates a NOT_FOUND domain error.
func NotFound(msg string) *AppError { return newError(CodeNotFound, msg) }

// Conflict creates a CONFLICT domain error.
func Conflict(msg string) *AppError { return newError(CodeConflict, msg) }

// BadRequest creates a BAD_REQUEST domain error.
func BadRequest(msg string) *AppError { return newError(CodeBadRequest, msg) }

// Forbidden creates a FORBIDDEN domain error.
func Forbidden(msg string) *AppError { return newError(CodeForbidden, msg) }

// Unauthorized creates an UNAUTHORIZED domain error.
func Unauthorized(msg string) *AppError { return newError(CodeUnauthorized, msg) }

// Internal creates an INTERNAL domain error.
func Internal(msg string, args ...any) *AppError {
	return newError(CodeInternal, fmt.Sprintf(msg, args...))
}
