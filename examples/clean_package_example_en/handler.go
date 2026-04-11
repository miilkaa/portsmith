package example

// handler.go — HTTP adapter (outermost layer).
//
// Rules for this layer:
//  1. Accepts dependencies ONLY through interfaces (UserService).
//  2. Knows about HTTP: gin.Context, status codes, JSON.
//  3. Must NOT know about SQL, gorm, or database/sql.
//  4. Must NOT contain business logic — only:
//     parse request → call service → build response.
//  5. Does not map service errors to HTTP codes itself — the error middleware
//     in pkg/server does this automatically via apperrors.
//
// Routes are registered via Routes() — the handler knows its own URLs.
// This is more convenient than registering routes in main.go.

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/miilkaa/portsmith/pkg/pagination"
	"github.com/miilkaa/portsmith/pkg/server"
)

// Handler implements the HTTP endpoints for user management.
type Handler struct {
	// service is injected through the interface, not as a concrete *Service.
	// The field name "service" follows the portsmith gen convention.
	service UserService
}

// NewHandler creates a new Handler.
func NewHandler(service UserService) *Handler {
	return &Handler{service: service}
}

// Routes registers the handler's endpoints in the provided Gin router group.
// Called from main.go or during server setup.
//
// Example:
//
//	v1 := srv.Router().Group("/api/v1")
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

// create handles POST /users.
func (h *Handler) create(c *gin.Context) {
	var req CreateUserRequest
	if err := server.BindAndValidate(c, &req); err != nil {
		return // BindAndValidate already wrote the error response
	}

	user, err := h.service.Create(c.Request.Context(), toCreateParams(req))
	if err != nil {
		_ = c.Error(err) // the error middleware will handle this
		return
	}

	c.JSON(http.StatusCreated, toResponse(user))
}

// getByID handles GET /users/:id.
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

// update handles PATCH /users/:id.
func (h *Handler) update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	var req UpdateUserRequest
	if err := server.BindAndValidate(c, &req); err != nil {
		return
	}

	// In a real project callerID comes from JWT/session middleware.
	// Using a stub here for simplicity.
	callerID := uint(1)

	user, err := h.service.Update(c.Request.Context(), id, toUpdateParams(req), callerID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toResponse(user))
}

// delete handles DELETE /users/:id.
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

// list handles GET /users with filter and pagination query parameters.
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

// parseID extracts and validates the :id URL parameter.
func parseID(c *gin.Context) (uint, error) {
	raw := c.Param("id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, err
	}
	return uint(id), nil
}

// parseListFilter reads filtering parameters from the query string.
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
