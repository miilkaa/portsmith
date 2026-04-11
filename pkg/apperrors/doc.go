// Package apperrors provides typed domain errors with HTTP status mapping.
//
// # Overview
//
// Define errors in the domain layer without any HTTP knowledge.
// The server middleware maps AppError types to HTTP status codes automatically.
//
// # Defining domain errors
//
//	package users
//
//	import "github.com/miilkaa/portsmith/pkg/apperrors"
//
//	var (
//	    ErrUserNotFound = apperrors.NotFound("user not found")
//	    ErrEmailTaken   = apperrors.Conflict("email already taken")
//	)
//
// # Error comparison
//
// Use errors.Is for sentinel error comparison — it works across error chains:
//
//	if errors.Is(err, users.ErrUserNotFound) { ... }
//
// # HTTP mapping
//
// The server error middleware calls apperrors.HTTPStatus automatically.
// Explicit use is rarely needed:
//
//	status := apperrors.HTTPStatus(err)  // 404, 409, 400, 403, 401, 500
//
// # Error codes
//
//	CodeNotFound     → 404
//	CodeConflict     → 409
//	CodeBadRequest   → 400
//	CodeForbidden    → 403
//	CodeUnauthorized → 401
//	CodeInternal     → 500
package apperrors
