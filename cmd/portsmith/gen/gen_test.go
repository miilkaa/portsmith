package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_callPatternErrorBlocksGeneration(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "portsmith.yaml", `lint:
  call_patterns:
    handler:
      allowed:
        - "*.svc.*"
      not_allowed:
        - "*.service.*"
`)
	writePackage(t, filepath.Join(root, "internal", "orders"))

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

	err = Run([]string{"internal/orders"})
	if err == nil || !strings.Contains(err.Error(), "call-pattern check failed") {
		t.Fatalf("expected call-pattern error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "orders", "ports.go")); !os.IsNotExist(err) {
		t.Fatalf("ports.go should not be generated when call-pattern fails, stat err: %v", err)
	}
}

func TestRun_callPatternWarningAllowsGeneration(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "portsmith.yaml", `lint:
  call_patterns:
    handler:
      not_allowed:
        - "*.service.*"
  rules:
    call-pattern:
      severity: warning
`)
	writePackage(t, filepath.Join(root, "internal", "orders"))

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

	if err := Run([]string{"internal/orders"}); err != nil {
		t.Fatalf("warning-severity call-pattern should not block generation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "orders", "ports.go")); err != nil {
		t.Fatalf("ports.go should be generated: %v", err)
	}
}

func TestRun_allowedServicePatternFeedsGeneration(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writeFile(t, root, "portsmith.yaml", `lint:
  call_patterns:
    handler:
      allowed:
        - "*.svc.*"
      not_allowed:
        - "*.service.*"
`)
	writeAllowedPackage(t, filepath.Join(root, "internal", "orders"))

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

	if err := Run([]string{"internal/orders"}); err != nil {
		t.Fatalf("allowed service pattern should feed generation: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "internal", "orders", "ports.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Do()") {
		t.Fatalf("generated ports.go should include method collected via allowed pattern:\n%s", raw)
	}
}

func writePackage(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "handler.go", `package orders
type Handler struct{}
func (h *Handler) Create() {
	h.service.Do()
}
`)
	writeFile(t, dir, "service.go", `package orders
type Service struct{}
func (s *Service) Do() {}
`)
	writeFile(t, dir, "repository.go", `package orders
type Repository struct{}
`)
}

func writeAllowedPackage(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "handler.go", `package orders
type Handler struct{}
func (h *Handler) Create() {
	h.svc.Do()
}
`)
	writeFile(t, dir, "service.go", `package orders
type Service struct{}
func (s *Service) Do() {}
`)
	writeFile(t, dir, "repository.go", `package orders
type Repository struct{}
`)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
