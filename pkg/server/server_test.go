package server_test

// server_test.go — контрактные тесты для pkg/server.
//
// Контракт:
//  1. New создаёт сервер с заданной конфигурацией.
//  2. /health возвращает 200 OK из коробки.
//  3. Recovery middleware перехватывает panic → 500.
//  4. Error middleware конвертирует apperrors в корректный HTTP-статус.
//  5. BindAndValidate возвращает 400 при невалидном теле.
//  6. RequestID middleware добавляет X-Request-ID в ответ.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/miilkaa/portsmith/pkg/apperrors"
	"github.com/miilkaa/portsmith/pkg/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestServer_healthEndpoint(t *testing.T) {
	srv := server.New(server.Config{Port: 0})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServer_recoveryMiddleware_panicReturns500(t *testing.T) {
	srv := server.New(server.Config{Port: 0})
	srv.Router().GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d", w.Code)
	}
}

func TestServer_errorMiddleware_appErrorMappedToHTTP(t *testing.T) {
	srv := server.New(server.Config{Port: 0})
	srv.Router().GET("/not-found", func(c *gin.Context) {
		_ = c.Error(apperrors.NotFound("user not found"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] == nil {
		t.Error("expected 'error' field in response body")
	}
}

func TestServer_errorMiddleware_unknownErrorReturns500(t *testing.T) {
	srv := server.New(server.Config{Port: 0})
	srv.Router().GET("/err", func(c *gin.Context) {
		_ = c.Error(http.ErrNoCookie)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestBindAndValidate_invalidJSON_returns400(t *testing.T) {
	type request struct {
		Email string `json:"email" binding:"required,email"`
	}

	srv := server.New(server.Config{Port: 0})
	srv.Router().POST("/test", func(c *gin.Context) {
		var req request
		if err := server.BindAndValidate(c, &req); err != nil {
			return
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	body := strings.NewReader(`{"email": "not-an-email"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.Header.Set("Content-Type", "application/json")
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestServer_requestIDMiddleware(t *testing.T) {
	srv := server.New(server.Config{Port: 0})
	srv.Router().GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}
