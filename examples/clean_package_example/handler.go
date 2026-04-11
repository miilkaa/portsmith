package example

// handler.go — HTTP-адаптер (внешний слой).
//
// Правила этого слоя:
//  1. Принимает зависимости ТОЛЬКО через интерфейсы (UserService).
//  2. Знает про HTTP: gin.Context, статус-коды, JSON.
//  3. НЕ знает про SQL, gorm, database/sql.
//  4. НЕ содержит бизнес-логики — только:
//     - распарсить запрос → вызвать сервис → сформировать ответ.
//  5. Ошибки сервиса не преобразует сам — error middleware сервера
//     делает это автоматически через apperrors.
//
// Маршруты регистрируются через Routes() — хендлер сам знает свои URL.
// Это удобнее, чем регистрировать маршруты в main.go.

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/miilkaa/portsmith/pkg/pagination"
	"github.com/miilkaa/portsmith/pkg/server"
)

// Handler реализует HTTP-обработчики для пользователей.
type Handler struct {
	// service — зависимость через интерфейс, не через *Service.
	// Имя поля "service" соответствует конвенции portsmith gen.
	service UserService
}

// NewHandler создаёт новый Handler.
func NewHandler(service UserService) *Handler {
	return &Handler{service: service}
}

// Routes регистрирует маршруты в переданной группе Gin.
// Вызывается из main.go или при настройке сервера.
//
// Пример:
//
//	v1 := srv.Group("/api/v1")
//	userHandler.Routes(v1)
func (h *Handler) Routes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.POST("", h.create)
		users.GET("", h.list)
		users.GET("/:id", h.getByID)
		users.PATCH("/:id", h.update)
		users.DELETE("/:id", h.delete)
	}
}

// create обрабатывает POST /users.
func (h *Handler) create(c *gin.Context) {
	var req CreateUserRequest
	if err := server.BindAndValidate(c, &req); err != nil {
		return // BindAndValidate уже записал ответ с ошибкой
	}

	user, err := h.service.Create(c.Request.Context(), toCreateParams(req))
	if err != nil {
		_ = c.Error(err) // error middleware обработает
		return
	}

	c.JSON(http.StatusCreated, toResponse(user))
}

// getByID обрабатывает GET /users/:id.
func (h *Handler) getByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toResponse(user))
}

// update обрабатывает PATCH /users/:id.
func (h *Handler) update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	var req UpdateUserRequest
	if err := server.BindAndValidate(c, &req); err != nil {
		return
	}

	// callerID в реальном проекте берётся из JWT/session middleware.
	// Здесь для простоты используем заглушку.
	callerID := uint(1)

	user, err := h.service.Update(c.Request.Context(), id, toUpdateParams(req), callerID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toResponse(user))
}

// delete обрабатывает DELETE /users/:id.
func (h *Handler) delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// list обрабатывает GET /users с query-параметрами фильтрации и пагинации.
func (h *Handler) list(c *gin.Context) {
	page := pagination.OffsetFromQuery(c.Request)
	filter := parseListFilter(c)

	users, total, err := h.service.List(c.Request.Context(), filter, page)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Items: toResponseList(users),
		Total: total,
		Page:  page.PageNumber(),
		Limit: page.Limit(),
	})
}

// parseID извлекает и валидирует ID из URL-параметра.
func parseID(c *gin.Context) (uint, error) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, err
	}
	return uint(id), nil
}

// parseListFilter извлекает параметры фильтрации из query-строки.
func parseListFilter(c *gin.Context) ListFilter {
	var filter ListFilter
	if role := c.Query("role"); role != "" {
		r := UserRole(role)
		filter.Role = &r
	}
	if active := c.Query("active"); active != "" {
		b := active == "true"
		filter.Active = &b
	}
	return filter
}
