package example

// dto.go — HTTP request and response structs.
//
// Rules for this layer:
//  1. DTOs (Data Transfer Objects) exist only for the HTTP boundary.
//  2. They carry json tags for serialisation.
//  3. They carry validate/binding tags for go-playground/validator.
//  4. They contain no business logic.
//  5. They are never passed directly into the service or repository.
//     The handler converts DTO → domain params (CreateParams, etc.)
//     before calling the service.
//
// Why a separate DTO layer instead of using Domain types directly?
//   - The HTTP contract can differ from the domain model
//     (e.g. password on create, computed fields on read).
//   - Internal structure can change without breaking the API contract.
//   - Validation tags do not pollute the domain layer.

// CreateUserRequest is the body for POST /users.
type CreateUserRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name"  binding:"required,min=2,max=100"`
	Role  UserRole `json:"role"  binding:"omitempty,oneof=user admin"`
}

// UpdateUserRequest is the body for PATCH /users/:id.
// Pointer fields allow distinguishing "field not sent" from "field sent as empty".
type UpdateUserRequest struct {
	Name   *string   `json:"name"   binding:"omitempty,min=2,max=100"`
	Role   *UserRole `json:"role"   binding:"omitempty,oneof=user admin"`
	Active *bool     `json:"active"`
}

// UserResponse is the user representation returned in API responses.
// Sensitive fields (passwords, tokens) are intentionally excluded.
type UserResponse struct {
	ID     uint     `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Role   UserRole `json:"role"`
	Active bool     `json:"active"`
}

// ListUsersResponse is the response for GET /users including pagination metadata.
type ListUsersResponse struct {
	Items []*UserResponse `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}
