package lintconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miilkaa/portsmith/internal/lintconfig"
)

func TestLoad_missingFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := lintconfig.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stack != "" {
		t.Fatalf("expected empty stack, got %q", cfg.Stack)
	}
}

func TestLoad_readsStackAndLint(t *testing.T) {
	dir := t.TempDir()
	content := `stack: chi-sqlx
lint:
  rules:
    test-files:
      severity: off
`
	if err := os.WriteFile(filepath.Join(dir, "portsmith.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := lintconfig.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stack != "chi-sqlx" {
		t.Fatalf("stack: got %q", cfg.Stack)
	}
	if cfg.Lint.RuleSeverity("test-files") != lintconfig.SeverityOff {
		t.Fatalf("test-files severity")
	}
}
