// Package chiserver provides a production-ready Chi HTTP server with:
//
//   - Chi middleware: RequestID, RealIP, Recoverer
//   - CORS headers
//   - JSON error responses via RespondError
//   - /health endpoint
//   - BindAndValidate for JSON + validator tags
//
// Usage:
//
//	srv := chiserver.New(chiserver.Config{Port: 8080})
//	r := srv.Router()
//	r.Mount("/api/v1", userHandler.Routes())
//	if err := srv.Run(); err != nil { ... }
package chiserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/miilkaa/portsmith/pkg/apperrors"
)

// Config holds server configuration.
type Config struct {
	Port int
}

// Server wraps a Chi router with portsmith defaults.
type Server struct {
	cfg    Config
	router chi.Router
	srv    *http.Server
}

// New creates a new Server with default middleware and /health.
func New(cfg Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)
	s := &Server{cfg: cfg, router: r}
	s.registerBuiltins()
	return s
}

// Router returns the Chi router for mounting domain routes.
func (s *Server) Router() chi.Router {
	return s.router
}

// Run listens and serves HTTP until the server stops.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	slog.Info("server starting", "addr", addr)
	s.srv = &http.Server{Addr: addr, Handler: s.router}
	return s.srv.ListenAndServe()
}

func (s *Server) registerBuiltins() {
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var validate = validator.New()

// BindAndValidate decodes JSON from the body and runs struct tag validation.
// On failure it writes 400 JSON and returns the error.
func BindAndValidate(w http.ResponseWriter, r *http.Request, dst any) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil && err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return err
	}
	if err := validate.Struct(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return err
	}
	return nil
}

// RespondError writes a JSON error body with status from apperrors.HTTPStatus.
func RespondError(w http.ResponseWriter, err error) {
	status := apperrors.HTTPStatus(err)
	body := map[string]any{"error": err.Error()}
	var appErr *apperrors.AppError
	if apperrors.As(err, &appErr) && len(appErr.Details()) > 0 {
		body["details"] = appErr.Details()
	}
	writeJSON(w, status, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
