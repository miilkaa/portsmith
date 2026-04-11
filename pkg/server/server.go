// Package server provides a production-ready Gin HTTP server with:
//
//   - Recovery middleware (panic → 500)
//   - Request ID middleware (generates X-Request-ID per request)
//   - CORS middleware
//   - Error handling middleware (converts apperrors → HTTP status + JSON body)
//   - /health endpoint out of the box
//   - BindAndValidate helper for JSON body parsing + validation
//
// Usage:
//
//	srv := server.New(server.Config{Port: 8080})
//
//	v1 := srv.Router().Group("/api/v1")
//	userHandler.Routes(v1)
//
//	if err := srv.Run(); err != nil {
//	    log.Fatal(err)
//	}
package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/miilkaa/portsmith/pkg/apperrors"
)

// Config holds server configuration.
type Config struct {
	// Port is the TCP port to listen on. 0 = OS-assigned (useful in tests).
	Port int

	// Mode sets the Gin mode: "debug", "release", "test". Defaults to "release".
	Mode string
}

// Server wraps a Gin engine with portsmith defaults.
type Server struct {
	cfg    Config
	engine *gin.Engine
}

// New creates a new Server with all default middleware applied.
func New(cfg Config) *Server {
	mode := cfg.Mode
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	engine := gin.New()

	s := &Server{cfg: cfg, engine: engine}
	s.applyMiddleware()
	s.registerBuiltins()
	return s
}

// Router returns the underlying *gin.Engine for route registration.
func (s *Server) Router() *gin.Engine {
	return s.engine
}

// Run starts the HTTP server. Blocks until the server exits.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	slog.Info("server starting", "addr", addr)
	return s.engine.Run(addr)
}

func (s *Server) applyMiddleware() {
	s.engine.Use(
		recoveryMiddleware(),
		requestIDMiddleware(),
		corsMiddleware(),
		errorMiddleware(),
	)
}

func (s *Server) registerBuiltins() {
	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// recoveryMiddleware catches panics and returns 500.
func recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic recovered", "error", fmt.Sprintf("%v", recovered))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
	})
}

// requestIDMiddleware generates a unique X-Request-ID for each request.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Header("X-Request-ID", id)
		c.Set("request_id", id)
		c.Next()
	}
}

// corsMiddleware adds permissive CORS headers.
// In production, restrict AllowOrigins to your actual domains.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// errorMiddleware inspects errors attached via c.Error() and writes
// a JSON error response with the appropriate HTTP status code.
// AppErrors are mapped via apperrors.HTTPStatus; others become 500.
func errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Use the last error as the primary one.
		err := c.Errors.Last().Err

		status := apperrors.HTTPStatus(err)
		body := gin.H{"error": err.Error()}

		// Attach structured details if present.
		var appErr *apperrors.AppError
		if apperrors.As(err, &appErr) && len(appErr.Details()) > 0 {
			body["details"] = appErr.Details()
		}

		c.JSON(status, body)
	}
}

// BindAndValidate binds the JSON request body and runs validator tag checks.
// On failure it writes a 400 response and returns the error.
// The handler must return immediately when err != nil.
//
//	var req CreateUserRequest
//	if err := server.BindAndValidate(c, &req); err != nil {
//	    return
//	}
func BindAndValidate(c *gin.Context, req any) error {
	if err := c.ShouldBindJSON(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return err
	}
	return nil
}
