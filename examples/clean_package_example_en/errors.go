package example

// Domain errors are declared here — in the domain layer.
//
// Rule: errors must not know about HTTP status codes. They describe
// a business situation. The error middleware in pkg/server maps
// apperrors types to HTTP statuses automatically:
//
//	apperrors.NotFound   → 404
//	apperrors.Conflict   → 409
//	apperrors.BadRequest → 400
//	apperrors.Forbidden  → 403
//
// Why sentinel variables instead of custom types?
// errors.Is works with sentinel values. This is the idiomatic Go pattern
// for domain errors and lets callers compare without type assertions.

import "github.com/miilkaa/portsmith/pkg/apperrors"

var (
	// ErrUserNotFound is returned when a user cannot be found by ID or email.
	ErrUserNotFound = apperrors.NotFound("user not found")

	// ErrEmailTaken is returned when creating a user with an email address
	// that is already registered.
	ErrEmailTaken = apperrors.Conflict("email already taken")

	// ErrCannotDeactivateSelf is returned when an administrator tries to
	// deactivate their own account.
	ErrCannotDeactivateSelf = apperrors.BadRequest("cannot deactivate your own account")

	// ErrInsufficientPermissions is returned when an operation requires
	// administrator privileges.
	ErrInsufficientPermissions = apperrors.Forbidden("insufficient permissions")
)
