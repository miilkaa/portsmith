package gen

import (
	"bytes"
	"io"
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
	writeAllowedPackage(t, filepath.Join(root, "internal", "customers"))

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

	err = Run([]string{"internal/orders", "internal/customers"})
	if err == nil || !strings.Contains(err.Error(), "call-pattern check failed") {
		t.Fatalf("expected call-pattern error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "orders", "ports.go")); !os.IsNotExist(err) {
		t.Fatalf("ports.go should not be generated when call-pattern fails, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "customers", "ports.go")); !os.IsNotExist(err) {
		t.Fatalf("ports.go should not be generated for any package when call-pattern fails, stat err: %v", err)
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

func TestRun_dryRunMultiplePackagesPrintsInInputOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writePackage(t, filepath.Join(root, "internal", "alpha"))
	writePackage(t, filepath.Join(root, "internal", "beta"))

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

	out := captureStdout(t, func() {
		if err := Run([]string{"--dry-run", "internal/beta", "internal/alpha"}); err != nil {
			t.Fatalf("dry-run should succeed: %v", err)
		}
	})
	beta := strings.Index(out, "=== internal/beta/ports.go ===")
	alpha := strings.Index(out, "=== internal/alpha/ports.go ===")
	if beta < 0 || alpha < 0 {
		t.Fatalf("expected dry-run output for both packages:\n%s", out)
	}
	if beta > alpha {
		t.Fatalf("dry-run output should follow input order:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "alpha", "ports.go")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not write alpha ports.go, stat err: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "beta", "ports.go")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not write beta ports.go, stat err: %v", err)
	}
}

func TestRun_generatesMultiplePackages(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writePackage(t, filepath.Join(root, "internal", "alpha"))
	writePackage(t, filepath.Join(root, "internal", "beta"))

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

	if err := Run([]string{"internal/alpha", "internal/beta"}); err != nil {
		t.Fatalf("generation should succeed: %v", err)
	}
	for _, dir := range []string{"alpha", "beta"} {
		raw, err := os.ReadFile(filepath.Join(root, "internal", dir, "ports.go"))
		if err != nil {
			t.Fatalf("%s ports.go should be generated: %v", dir, err)
		}
		if !strings.Contains(string(raw), "Do()") {
			t.Fatalf("%s ports.go should include Do():\n%s", dir, raw)
		}
	}
}

func TestRun_verbosePrintsProgressToStderr(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writePackage(t, filepath.Join(root, "internal", "alpha"))
	writePackage(t, filepath.Join(root, "internal", "beta"))

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

	errOut := captureStderr(t, func() {
		if err := Run([]string{"-v", "internal/alpha", "internal/beta"}); err != nil {
			t.Fatalf("verbose generation should succeed: %v", err)
		}
	})
	for _, want := range []string{
		"portsmith gen: workers=",
		"packages=2",
		"portsmith gen: call-pattern start internal/alpha",
		"portsmith gen: generate start internal/alpha",
		"portsmith gen: completed in ",
	} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("verbose stderr missing %q:\n%s", want, errOut)
		}
	}
}

func TestRun_withoutVerboseDoesNotPrintProgress(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\ngo 1.22\n")
	writePackage(t, filepath.Join(root, "internal", "alpha"))

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

	errOut := captureStderr(t, func() {
		if err := Run([]string{"internal/alpha"}); err != nil {
			t.Fatalf("generation should succeed: %v", err)
		}
	})
	if strings.Contains(errOut, "portsmith gen:") {
		t.Fatalf("progress should be hidden without -v, got:\n%s", errOut)
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
