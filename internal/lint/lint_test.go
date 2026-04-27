package lint_test

import (
	"os"
	"path/filepath"
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

func index(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
