package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miilkaa/portsmith/internal/project"
)

func TestLoad_missingFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := project.Load(dir)
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
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Stack != "chi-sqlx" {
		t.Fatalf("stack: got %q", cfg.Stack)
	}
	if cfg.Lint.RuleSeverity("test-files") != project.SeverityOff {
		t.Fatalf("test-files severity")
	}
}

func TestRuleSeverity_loggerRules_optIn(t *testing.T) {
	cfg := project.Config{}
	if cfg.Lint.RuleSeverity("logger-no-other") != project.SeverityOff {
		t.Fatalf("logger-no-other without config should be off")
	}
	cfg.Lint.Logger.Allowed = "log/slog"
	if cfg.Lint.RuleSeverity("logger-no-fmt-print") != project.SeverityError {
		t.Fatalf("logger rules should be error when logger.allowed is set")
	}
}

func TestRuleSeverity_loggerRules_explicitOverride(t *testing.T) {
	cfg := project.Config{
		Lint: project.LintConfig{
			Logger: project.LoggerConfig{Allowed: "log/slog"},
			Rules: map[string]project.RuleConfig{
				"logger-no-init": {Severity: "off"},
			},
		},
	}
	if cfg.Lint.RuleSeverity("logger-no-init") != project.SeverityOff {
		t.Fatalf("explicit rules entry should win")
	}
	if cfg.Lint.RuleSeverity("logger-no-other") != project.SeverityError {
		t.Fatalf("other logger rules should stay error")
	}
}

func TestRuleSeverity_callPattern_optIn(t *testing.T) {
	cfg := project.Config{}
	if cfg.Lint.RuleSeverity("call-pattern") != project.SeverityOff {
		t.Fatalf("call-pattern without call_patterns should be off")
	}
	cfg.Lint.CallPatterns.Handler.Allowed = []string{"*.svc.*"}
	if cfg.Lint.RuleSeverity("call-pattern") != project.SeverityOff {
		t.Fatalf("call-pattern should stay off when only allowed is set")
	}
	cfg.Lint.CallPatterns.Handler.NotAllowed = []string{"*.service.*"}
	if cfg.Lint.RuleSeverity("call-pattern") != project.SeverityError {
		t.Fatalf("call-pattern should be error when not_allowed is set")
	}
}

func TestRuleSeverity_callPattern_explicitOverride(t *testing.T) {
	cfg := project.Config{
		Lint: project.LintConfig{
			CallPatterns: project.CallPatternsConfig{
				Handler: project.LayerCallConfig{
					NotAllowed: []string{"*.service.*"},
				},
			},
			Rules: map[string]project.RuleConfig{
				"call-pattern": {Severity: "off"},
			},
		},
	}
	if cfg.Lint.RuleSeverity("call-pattern") != project.SeverityOff {
		t.Fatalf("explicit rules entry should win over call_patterns")
	}
}

func TestLoad_callPatterns(t *testing.T) {
	dir := t.TempDir()
	content := `stack: chi-sqlx
lint:
  call_patterns:
    handler:
      allowed:
        - "*.svc.*"
      not_allowed:
        - "*.service.*"
    service:
      not_allowed:
        - "*.repository.*"
`
	if err := os.WriteFile(filepath.Join(dir, "portsmith.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Lint.CallPatterns.Handler.Allowed) != 1 || cfg.Lint.CallPatterns.Handler.Allowed[0] != "*.svc.*" {
		t.Fatalf("handler allowed: %#v", cfg.Lint.CallPatterns.Handler.Allowed)
	}
	if len(cfg.Lint.CallPatterns.Handler.NotAllowed) != 1 || cfg.Lint.CallPatterns.Handler.NotAllowed[0] != "*.service.*" {
		t.Fatalf("handler not_allowed: %#v", cfg.Lint.CallPatterns.Handler.NotAllowed)
	}
	if len(cfg.Lint.CallPatterns.Service.NotAllowed) != 1 {
		t.Fatalf("service not_allowed: %#v", cfg.Lint.CallPatterns.Service.NotAllowed)
	}
}
