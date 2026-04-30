package lint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/lintconfig"
)

func TestViolations_rule5_repositoryPortInHandler(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "ports.go", `package orders
type OrdersRepository interface { Get() }
type OrdersService interface { Do() }
`)
	write(t, dir, "repository.go", `package orders
type Repository struct{}
`)
	write(t, dir, "service.go", `package orders
type Service struct{}
`)
	write(t, dir, "handler.go", `package orders
type Handler struct {
	repo OrdersRepository
}
`)
	write(t, dir, "handler_test.go", `package orders
import "testing"
func TestX(t *testing.T) {}
`)
	write(t, dir, "service_test.go", `package orders
import "testing"
func TestY(t *testing.T) {}
`)

	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range vs {
		if v.Rule == "layer-dependency" && contains(v.Message, "repository-layer") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected rule5 violation, got %#v", vs)
	}
}

func TestViolations_rule7_maxLines(t *testing.T) {
	dir := t.TempDir()
	body := "package p\nconst x = 1\n"
	write(t, dir, "tiny.go", body)
	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			MaxLines: []lintconfig.FileSizeRule{
				{Pattern: "*.go", Limit: 1},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Rule != "file-size" {
		t.Fatalf("expected one rule7 violation, got %#v", vs)
	}
}

func TestViolations_loggerRules_disabledWithoutConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "x.go", `package p
import "log"
func f() { log.Print("x") }
`)
	cfg := lintconfig.Config{}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if stringsHasPrefix(v.Rule, "logger-") {
			t.Fatalf("unexpected logger violation without lint.logger.allowed: %v", v)
		}
	}
}

func TestViolations_loggerRules_importAndFmt(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "x.go", `package p
import "log"
func f() { fmt.Printf("%s", "x") }
`)
	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Logger: lintconfig.LoggerConfig{Allowed: "log/slog"},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	var hasOther, hasFmt bool
	for _, v := range vs {
		if v.Rule == "logger-no-other" {
			hasOther = true
		}
		if v.Rule == "logger-no-fmt-print" {
			hasFmt = true
		}
	}
	if !hasOther || !hasFmt {
		t.Fatalf("expected logger-no-other and logger-no-fmt-print, got %#v", vs)
	}
}

func TestViolations_loggerRules_slogNew(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "x.go", `package p
import "log/slog"
func f() { _ = slog.New(nil) }
`)
	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Logger: lintconfig.LoggerConfig{Allowed: "log/slog"},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range vs {
		if v.Rule == "logger-no-init" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected logger-no-init, got %#v", vs)
	}
}

func TestViolations_contextFirstExemptsWiringMethods(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "service.go", `package orders
type Service struct{}
type Client interface{}
type Deps struct{}
func (s *Service) SetClient(client Client) {}
func (s *Service) WithDeps(deps *Deps) *Service { return s }
func (s *Service) Do(input string) error { return nil }
`)
	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}

	var contextFirst []lint.Violation
	for _, v := range vs {
		if v.Rule == "context-first" {
			contextFirst = append(contextFirst, v)
		}
	}
	if len(contextFirst) != 1 || !contains(contextFirst[0].Message, "Do") {
		t.Fatalf("expected only Do to violate context-first, got %#v", contextFirst)
	}
}

func TestViolations_callPattern_handlerNotAllowed(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "ports.go", `package orders
type OrdersRepository interface { Get() }
type OrdersService interface { Do() }
`)
	write(t, dir, "repository.go", `package orders
type Repository struct{}
`)
	write(t, dir, "service.go", `package orders
type Service struct{}
`)
	write(t, dir, "handler.go", `package orders
type Handler struct{}
func (h *Handler) Create() {
	h.service.Do()
}
`)
	write(t, dir, "handler_test.go", `package orders
import "testing"
func TestX(t *testing.T) {}
`)
	write(t, dir, "service_test.go", `package orders
import "testing"
func TestY(t *testing.T) {}
`)

	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
			CallPatterns: lintconfig.CallPatternsConfig{
				Handler: lintconfig.LayerCallConfig{
					Allowed:    []string{"*.svc.*"},
					NotAllowed: []string{"*.service.*"},
				},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range vs {
		if v.Rule == "call-pattern" && contains(v.Message, "h.service.Do") && contains(v.Message, "*.svc.*") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected call-pattern violation, got %#v", vs)
	}
}

func TestViolations_callPattern_handlerAllowedNoViolation(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "ports.go", `package orders
type OrdersRepository interface { Get() }
type OrdersService interface { Do() }
`)
	write(t, dir, "repository.go", `package orders
type Repository struct{}
`)
	write(t, dir, "service.go", `package orders
type Service struct{}
`)
	write(t, dir, "handler.go", `package orders
type Handler struct{}
func (h *Handler) Create() {
	h.svc.Do()
}
`)
	write(t, dir, "handler_test.go", `package orders
import "testing"
func TestX(t *testing.T) {}
`)
	write(t, dir, "service_test.go", `package orders
import "testing"
func TestY(t *testing.T) {}
`)

	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
			CallPatterns: lintconfig.CallPatternsConfig{
				Handler: lintconfig.LayerCallConfig{
					NotAllowed: []string{"*.service.*"},
				},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if v.Rule == "call-pattern" {
			t.Fatalf("unexpected call-pattern violation: %v", v)
		}
	}
}

func TestViolations_callPattern_wildcardReceiver(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "ports.go", `package orders
type OrdersRepository interface { Get() }
type OrdersService interface { Do() }
`)
	write(t, dir, "repository.go", `package orders
type Repository struct{}
`)
	write(t, dir, "service.go", `package orders
type Service struct{}
`)
	write(t, dir, "handler.go", `package orders
type Handler struct{}
func (h *Handler) A() { h.svc.Do() }
func (handler *Handler) B() { handler.svc.List() }
`)
	write(t, dir, "handler_test.go", `package orders
import "testing"
func TestX(t *testing.T) {}
`)
	write(t, dir, "service_test.go", `package orders
import "testing"
func TestY(t *testing.T) {}
`)

	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
			CallPatterns: lintconfig.CallPatternsConfig{
				Handler: lintconfig.LayerCallConfig{
					NotAllowed: []string{"*.service.*"},
				},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if v.Rule == "call-pattern" {
			t.Fatalf("unexpected call-pattern violation: %v", v)
		}
	}
}

func TestViolations_nolintSuppressesRule(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "handler.go", `package orders
//nolint:portsmith:handler-no-db
import "gorm.io/gorm"
type Handler struct{ db *gorm.DB }
`)
	write(t, dir, "service.go", `package orders`)
	write(t, dir, "repository.go", `package orders`)
	write(t, dir, "ports.go", `package orders`)

	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Rules: map[string]lintconfig.RuleConfig{
				"test-files": {Severity: "off"},
			},
		},
	}
	vs, err := lint.Violations(dir, cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		if v.Rule == "handler-no-db" {
			t.Fatalf("handler-no-db should be suppressed, got %#v", vs)
		}
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (index(s, sub) >= 0)
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.HasPrefix(s, prefix)
}

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
