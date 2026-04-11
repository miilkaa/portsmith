package example

// mappers.go — преобразование между слоями.
//
// Правила:
//  1. Маперы — чистые функции, без побочных эффектов.
//  2. Направление: DTO → доменные параметры (в handler, перед вызовом сервиса).
//  3. Направление: Domain → DTO (в handler, перед отправкой ответа).
//  4. Маперы не содержат бизнес-логики — только поле-в-поле.
//
// Почему не конструкторы на DTO/Domain?
// Чистые функции проще тестировать и они явно показывают трансформацию данных.

// toCreateParams преобразует HTTP-запрос в параметры для сервиса.
func toCreateParams(req CreateUserRequest) CreateParams {
	return CreateParams{
		Email: req.Email,
		Name:  req.Name,
		Role:  req.Role,
	}
}

// toUpdateParams преобразует HTTP-запрос в параметры для сервиса.
func toUpdateParams(req UpdateUserRequest) UpdateParams {
	return UpdateParams{
		Name:   req.Name,
		Role:   req.Role,
		Active: req.Active,
	}
}

// toResponse преобразует доменную сущность в DTO для HTTP-ответа.
func toResponse(u *User) *UserResponse {
	return &UserResponse{
		ID:     u.ID,
		Email:  u.Email,
		Name:   u.Name,
		Role:   u.Role,
		Active: u.Active,
	}
}

// toResponseList преобразует список доменных сущностей в список DTO.
func toResponseList(users []*User) []*UserResponse {
	result := make([]*UserResponse, len(users))
	for i, u := range users {
		result[i] = toResponse(u)
	}
	return result
}
