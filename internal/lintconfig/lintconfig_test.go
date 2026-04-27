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

func TestRuleSeverity_loggerRules_optIn(t *testing.T) {
	cfg := lintconfig.Config{}
	if cfg.Lint.RuleSeverity("logger-no-other") != lintconfig.SeverityOff {
		t.Fatalf("logger-no-other without config should be off")
	}
	cfg.Lint.Logger.Allowed = "log/slog"
	if cfg.Lint.RuleSeverity("logger-no-fmt-print") != lintconfig.SeverityError {
		t.Fatalf("logger rules should be error when logger.allowed is set")
	}
}

func TestRuleSeverity_loggerRules_explicitOverride(t *testing.T) {
	cfg := lintconfig.Config{
		Lint: lintconfig.LintConfig{
			Logger: lintconfig.LoggerConfig{Allowed: "log/slog"},
			Rules: map[string]lintconfig.RuleConfig{
				"logger-no-init": {Severity: "off"},
			},
		},
	}
	if cfg.Lint.RuleSeverity("logger-no-init") != lintconfig.SeverityOff {
		t.Fatalf("explicit rules entry should win")
	}
	if cfg.Lint.RuleSeverity("logger-no-other") != lintconfig.SeverityError {
		t.Fatalf("other logger rules should stay error")
	}
}
