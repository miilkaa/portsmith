package example

// dto.go — структуры запросов и ответов HTTP-слоя.
//
// Правила этого слоя:
//  1. DTO — Data Transfer Objects. Они существуют только для HTTP.
//  2. Содержат json-теги для сериализации.
//  3. Содержат validate-теги для go-playground/validator.
//  4. Не содержат бизнес-логики.
//  5. Никогда не передаются в service или repository напрямую.
//     Handler преобразует DTO → доменные параметры (CreateParams и т.д.).
//
// Зачем отдельный слой DTO, а не использовать Domain напрямую?
//   - HTTP-контракт может отличаться от доменной модели (например, password при создании).
//   - Можно менять внутреннюю структуру без изменения API.
//   - validate-теги не засоряют доменный слой.

// CreateUserRequest — тело POST /users.
type CreateUserRequest struct {
	Email string   `json:"email" binding:"required,email"`
	Name  string   `json:"name"  binding:"required,min=2,max=100"`
	Role  UserRole `json:"role"  binding:"omitempty,oneof=user admin"`
}

// UpdateUserRequest — тело PATCH /users/:id.
// Указатели позволяют отличить "поле не передано" от "передано пустое значение".
type UpdateUserRequest struct {
	Name   *string   `json:"name"   binding:"omitempty,min=2,max=100"`
	Role   *UserRole `json:"role"   binding:"omitempty,oneof=user admin"`
	Active *bool     `json:"active"`
}

// UserResponse — представление пользователя в API-ответе.
// Не включает чувствительные данные (пароли, токены).
type UserResponse struct {
	ID     uint     `json:"id"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Role   UserRole `json:"role"`
	Active bool     `json:"active"`
}

// ListUsersResponse — ответ на GET /users со списком и метаданными пагинации.
type ListUsersResponse struct {
	Items  []*UserResponse `json:"items"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}
