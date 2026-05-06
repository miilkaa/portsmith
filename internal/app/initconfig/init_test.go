package initconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	initconfig "github.com/miilkaa/portsmith/internal/app/initconfig"
)

func TestDetectLang_russianLocale(t *testing.T) {
	t.Setenv("LANG", "ru_RU.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	if got := initconfig.DetectLang(); got != "ru" {
		t.Fatalf("DetectLang() = %q, want ru", got)
	}
}

func TestDetectLang_englishLocale(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	if got := initconfig.DetectLang(); got != "en" {
		t.Fatalf("DetectLang() = %q, want en", got)
	}
}

func TestDetectLang_LC_ALL_precedence(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	if got := initconfig.DetectLang(); got != "ru" {
		t.Fatalf("DetectLang() = %q, want ru (LC_ALL first)", got)
	}
}

func TestDetectLang_emptyDefaultsToEnglish(t *testing.T) {
	t.Setenv("LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	if got := initconfig.DetectLang(); got != "en" {
		t.Fatalf("DetectLang() = %q, want en", got)
	}
}

func TestRun_unexpectedArg(t *testing.T) {
	err := initconfig.Run([]string{"extra"})
	if err == nil {
		t.Fatal("expected error for unexpected argument")
	}
	if !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("error should mention unexpected args: %v", err)
	}
}

func TestRunWithOptions_writesPortsmithYAML(t *testing.T) {
	dir := t.TempDir()
	answers := &initconfig.WizardAnswers{
		Stack:           "chi-sqlx",
		LoggerImport:    "log/slog",
		MaxLinesLimit:   300,
		MaxMethodsLimit: 15,
		WiringMode:      "default",
	}
	if err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Lang: "en", Answers: answers}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "portsmith.yaml")
	raw := readFile(t, path)
	if !strings.Contains(raw, "stack: chi-sqlx") {
		t.Fatalf("missing stack in:\n%s", raw)
	}
	if !strings.Contains(raw, "allowed: log/slog") {
		t.Fatalf("missing logger in:\n%s", raw)
	}
	if !strings.Contains(raw, "limit: 300") {
		t.Fatalf("missing max_lines in:\n%s", raw)
	}
	if !strings.Contains(raw, "limit: 15") {
		t.Fatalf("missing max_methods in:\n%s", raw)
	}
	if !strings.Contains(raw, "wire.go") || !strings.Contains(raw, "app.go") {
		t.Fatalf("missing wiring files in:\n%s", raw)
	}
}

func TestRunWithOptions_moduleCommentFromGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/acme/demo\n\ngo 1.22\n")
	answers := minimalAnswers()
	if err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Lang: "en", Answers: answers}); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(dir, "portsmith.yaml"))
	if !strings.Contains(raw, "# module: github.com/acme/demo") {
		t.Fatalf("expected module comment in:\n%s", raw)
	}
}

func TestRunWithOptions_existingFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "portsmith.yaml")
	writeFile(t, path, "stack: chi-sqlx\n")
	err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Answers: minimalAnswers()})
	if err == nil {
		t.Fatal("expected error when portsmith.yaml exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunWithOptions_forceOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "portsmith.yaml")
	writeFile(t, path, "stack: gin-gorm\n")
	answers := minimalAnswers()
	if err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Force: true, Answers: answers}); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, path)
	if !strings.Contains(raw, "stack: chi-sqlx") {
		t.Fatalf("expected overwrite to chi-sqlx:\n%s", raw)
	}
}

func TestRunWithOptions_skipLoggerAndLimits(t *testing.T) {
	dir := t.TempDir()
	answers := &initconfig.WizardAnswers{
		Stack:           "chi-sqlx",
		LoggerImport:    "",
		MaxLinesLimit:   0,
		MaxMethodsLimit: 0,
		WiringMode:      "skip",
	}
	if err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Answers: answers}); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(dir, "portsmith.yaml"))
	if strings.Contains(raw, "\n  logger:\n") {
		t.Fatalf("did not expect active logger block (uncommented):\n%s", raw)
	}
	if !strings.Contains(raw, "# max_lines:") {
		t.Fatalf("expected commented max_lines:\n%s", raw)
	}
	if !strings.Contains(raw, "# max_methods:") {
		t.Fatalf("expected commented max_methods:\n%s", raw)
	}
	if !strings.Contains(raw, "# wiring:") {
		t.Fatalf("expected commented wiring:\n%s", raw)
	}
}

func TestRunWithOptions_customWiring(t *testing.T) {
	dir := t.TempDir()
	answers := &initconfig.WizardAnswers{
		Stack:           "gin-gorm",
		WiringMode:      "custom",
		WiringFiles:     "cmd/wire.go , app/wiring.go",
		MaxLinesLimit:   150,
		MaxMethodsLimit: 10,
	}
	if err := initconfig.RunWithOptions(initconfig.Options{Dir: dir, Answers: answers}); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, filepath.Join(dir, "portsmith.yaml"))
	if !strings.Contains(raw, "stack: gin-gorm") {
		t.Fatalf("expected gin-gorm:\n%s", raw)
	}
	if !strings.Contains(raw, "- cmd/wire.go") || !strings.Contains(raw, "- app/wiring.go") {
		t.Fatalf("expected custom wiring list:\n%s", raw)
	}
}

func minimalAnswers() *initconfig.WizardAnswers {
	return &initconfig.WizardAnswers{
		Stack:           "chi-sqlx",
		MaxLinesLimit:   300,
		MaxMethodsLimit: 0,
		WiringMode:      "default",
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}
