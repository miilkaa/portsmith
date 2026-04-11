package example

// mappers.go — conversions between layers.
//
// Rules:
//  1. Mappers are pure functions with no side effects.
//  2. Direction DTO → domain params: called in the handler before invoking the service.
//  3. Direction Domain → DTO: called in the handler before sending the response.
//  4. Mappers contain no business logic — field-to-field only.
//
// Why pure functions instead of constructors on DTO/Domain?
// Pure functions are easier to test and make the transformation explicit.

// toCreateParams converts an HTTP request into service-layer parameters.
func toCreateParams(req CreateUserRequest) CreateParams {
	return CreateParams{
		Email: req.Email,
		Name:  req.Name,
		Role:  req.Role,
	}
}

// toUpdateParams converts an HTTP request into service-layer parameters.
func toUpdateParams(req UpdateUserRequest) UpdateParams {
	return UpdateParams{
		Name:   req.Name,
		Role:   req.Role,
		Active: req.Active,
	}
}

// toResponse converts a domain entity into an HTTP response DTO.
func toResponse(u *User) *UserResponse {
	return &UserResponse{
		ID:     u.ID,
		Email:  u.Email,
		Name:   u.Name,
		Role:   u.Role,
		Active: u.Active,
	}
}

// toResponseList converts a slice of domain entities into a slice of DTOs.
func toResponseList(users []*User) []*UserResponse {
	result := make([]*UserResponse, len(users))
	for i, u := range users {
		result[i] = toResponse(u)
	}
	return result
}
