package check_test

// check_test.go — контрактные тесты для cmd/portsmith/check.
//
// Контракт линтера:
//  1. Violations обнаруживает импорт gorm в handler.go.
//  2. Violations обнаруживает импорт net/http в service.go.
//  3. Violations обнаруживает конкретный тип в поле struct Handler/Service.
//  4. Violations возвращает пустой срез для корректного пакета.
//  5. ViolationError содержит файл, строку и описание.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miilkaa/portsmith/cmd/portsmith/check"
)

func setupPackage(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestCheck_handlerImportsGorm_violation(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"handler.go": `package orders
import "gorm.io/gorm"
type Handler struct{ db *gorm.DB }`,
		"service.go":    `package orders`,
		"repository.go": `package orders`,
		"ports.go":      `package orders`,
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsMessage(violations, "handler") || !containsMessage(violations, "gorm") {
		t.Errorf("expected gorm-in-handler violation, got: %v", violations)
	}
}

func TestCheck_serviceImportsChi_violation(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"service.go": `package orders
import "github.com/go-chi/chi/v5"
var _ chi.Router`,
		"handler.go":    `package orders`,
		"repository.go": `package orders`,
		"ports.go":      `package orders`,
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsMessage(violations, "service") || !containsMessage(violations, "chi") {
		t.Errorf("expected chi-in-service violation, got: %v", violations)
	}
}

func TestCheck_serviceImportsHTTP_violation(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"service.go": `package orders
import "net/http"
func (s *Service) Handle(w http.ResponseWriter) {}`,
		"handler.go":    `package orders`,
		"repository.go": `package orders`,
		"ports.go":      `package orders`,
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsMessage(violations, "service") || !containsMessage(violations, "net/http") {
		t.Errorf("expected net/http-in-service violation, got: %v", violations)
	}
}

func TestCheck_handlerConcreteServiceField_violation(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"handler.go": `package orders
type Handler struct {
	service *Service
}`,
		"service.go":    `package orders`,
		"repository.go": `package orders`,
		"ports.go":      `package orders`,
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsMessage(violations, "concrete") {
		t.Errorf("expected concrete-type violation, got: %v", violations)
	}
}

func TestCheck_missingPortsGo_violation(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"handler.go":    `package orders`,
		"service.go":    `package orders`,
		"repository.go": `package orders`,
		// No ports.go
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsMessage(violations, "ports.go") {
		t.Errorf("expected missing-ports.go violation, got: %v", violations)
	}
}

func TestCheck_cleanPackage_noViolations(t *testing.T) {
	dir := setupPackage(t, map[string]string{
		"handler.go": `package orders
import "context"
type Handler struct{ service OrdersService }
func (h *Handler) Do(ctx context.Context) {}`,
		"service.go": `package orders
import "context"
type Service struct{ repo OrdersRepository }
func (s *Service) Do(ctx context.Context) {}`,
		"repository.go": `package orders
import "gorm.io/gorm"
type Repository struct{ db *gorm.DB }`,
		"ports.go": `package orders
import "context"
type OrdersRepository interface{ FindByID(ctx context.Context, id uint) error }
type OrdersService interface{ Do(ctx context.Context) }`,
	})

	violations, err := check.Violations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations for clean package, got: %v", violations)
	}
}

func containsMessage(violations []check.Violation, substr string) bool {
	for _, v := range violations {
		if contains(v.Message, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
