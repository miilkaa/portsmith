package check_test

// check_test.go — integration tests for portsmith check (lint.Violations).

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	checkcmd "github.com/miilkaa/portsmith/cmd/portsmith/check"
	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/lintconfig"
)

func testLintConfig() lintconfig.Config {
	return lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
		},
	}
}

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

	violations, err := lint.Violations(dir, testLintConfig(), dir)
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

	violations, err := lint.Violations(dir, testLintConfig(), dir)
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

	violations, err := lint.Violations(dir, testLintConfig(), dir)
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

	violations, err := lint.Violations(dir, testLintConfig(), dir)
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
	})

	violations, err := lint.Violations(dir, testLintConfig(), dir)
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

	violations, err := lint.Violations(dir, testLintConfig(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected no violations for clean package, got: %v", violations)
	}
}

func TestRun_multiplePackagesPrintsSortedViolations(t *testing.T) {
	root := t.TempDir()
	writeCheckFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeCheckFile(t, root, "portsmith.yaml", `lint:
  rules:
    test-files:
      severity: off
    service-no-http:
      severity: warning
`)
	handlerDir := filepath.Join(root, "internal", "handlerbad")
	writeCheckFile(t, handlerDir, "handler.go", `package handlerbad
import "gorm.io/gorm"
type Handler struct{ db *gorm.DB }
`)
	writeCheckFile(t, handlerDir, "service.go", `package handlerbad`)
	writeCheckFile(t, handlerDir, "repository.go", `package handlerbad`)
	writeCheckFile(t, handlerDir, "ports.go", `package handlerbad`)

	serviceDir := filepath.Join(root, "internal", "servicebad")
	writeCheckFile(t, serviceDir, "handler.go", `package servicebad`)
	writeCheckFile(t, serviceDir, "service.go", `package servicebad
import (
	"context"
	"net/http"
)
func (s *Service) Handle(ctx context.Context, w http.ResponseWriter) {}
`)
	writeCheckFile(t, serviceDir, "repository.go", `package servicebad`)
	writeCheckFile(t, serviceDir, "ports.go", `package servicebad`)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()

	out, err := captureCheckStdout(func() error {
		return checkcmd.Run([]string{"internal/servicebad", "internal/handlerbad"})
	})
	if err == nil || !strings.Contains(err.Error(), "1 error violation(s)") {
		t.Fatalf("expected one error violation, got err=%v out=%s", err, out)
	}
	warnIdx := strings.Index(out, "warning [service-no-http]")
	errIdx := strings.Index(out, "error   [handler-no-db]")
	if warnIdx < 0 || errIdx < 0 {
		t.Fatalf("expected warning and error output, got:\n%s", out)
	}
	if warnIdx > errIdx {
		t.Fatalf("warnings should print before errors, got:\n%s", out)
	}
	if !strings.Contains(out, "2 violation(s)") ||
		!strings.Contains(out, "handler-no-db: 1") ||
		!strings.Contains(out, "service-no-http: 1") {
		t.Fatalf("expected deterministic summary, got:\n%s", out)
	}
}

func TestRun_missingPackageReturnsError(t *testing.T) {
	root := t.TempDir()
	writeCheckFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()

	_, err = captureCheckStdout(func() error {
		return checkcmd.Run([]string{"internal/missing"})
	})
	if err == nil || !strings.Contains(err.Error(), "internal/missing") {
		t.Fatalf("expected missing package error, got %v", err)
	}
}

func containsMessage(violations []lint.Violation, substr string) bool {
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

func writeCheckFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func captureCheckStdout(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	runErr := fn()

	closeErr := w.Close()
	if closeErr != nil {
		return "", closeErr
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	if err := r.Close(); err != nil {
		return "", err
	}
	return buf.String(), runErr
}
