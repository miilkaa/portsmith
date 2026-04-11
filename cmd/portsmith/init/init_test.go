package initcmd_test

// init_test.go — contract tests for portsmith init command.
//
// TDD Red phase: all tests must fail before implementation exists.
// The tests define the public contract of CheckDirty and RunWithFS.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	initcmd "github.com/miilkaa/portsmith/cmd/portsmith/init"
)

// fakeExamplesFS returns a minimal fake embed.FS for testing.
// Mirrors the structure of the real examples/ directory.
func fakeExamplesFS() fstest.MapFS {
	return fstest.MapFS{
		"examples/clean_package_example_en/domain.go": &fstest.MapFile{Data: []byte("package example\n")},
		"examples/clean_package_example_ru/domain.go": &fstest.MapFile{Data: []byte("package example\n")},
	}
}

// --- CheckDirty ---

func TestCheckDirty_emptyDir(t *testing.T) {
	dir := t.TempDir()
	blocking, err := initcmd.CheckDirty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocking) > 0 {
		t.Errorf("expected no blocking files, got %v", blocking)
	}
}

func TestCheckDirty_harmlessFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module myapp\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "go.sum"), "")
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(dir, "README.md"), "# hello\n")
	writeFile(t, filepath.Join(dir, ".env.example"), "PORT=8080\n")
	writeFile(t, filepath.Join(dir, "Makefile"), "test:\n\tgo test ./...\n")

	blocking, err := initcmd.CheckDirty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocking) > 0 {
		t.Errorf("expected no blocking files, got %v", blocking)
	}
}

func TestCheckDirty_rootGoFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n")

	blocking, err := initcmd.CheckDirty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocking) == 0 {
		t.Error("expected blocking files, got none")
	}
}

func TestCheckDirty_internalGoFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "internal", "orders"), 0o755)
	writeFile(t, filepath.Join(dir, "internal", "orders", "domain.go"), "package orders\n")

	blocking, err := initcmd.CheckDirty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocking) == 0 {
		t.Error("expected blocking files, got none")
	}
}

func TestCheckDirty_cmdGoFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cmd", "server"), 0o755)
	writeFile(t, filepath.Join(dir, "cmd", "server", "main.go"), "package main\n")

	blocking, err := initcmd.CheckDirty(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(blocking) == 0 {
		t.Error("expected blocking files for cmd/server/main.go")
	}
}

// --- Run (arg parsing) ---

func TestRun_missingAppName_returnsError(t *testing.T) {
	err := initcmd.Run([]string{})
	if err == nil {
		t.Error("expected error when app-name is missing")
	}
}

func TestRun_flagsAfterAppName(t *testing.T) {
	// Verify that flags placed AFTER the positional app-name are parsed.
	// We test this via RunWithFS in a temp dir rather than calling Run()
	// (which uses os.Getwd() and would create files in the source tree).
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent, Module: "github.com/x/myapp"}

	// Simulate what Run() does after splitFlagsAndPositionals reorders args:
	// "myapp --module github.com/x/myapp" → module must be picked up.
	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(parent, "myapp", "go.mod"))
	if !strings.Contains(content, "github.com/x/myapp") {
		t.Errorf("module from flag not applied, got:\n%s", content)
	}
}

// --- RunWithFS ---

func TestRunWithFS_emptyDir_createsFiles(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appDir := filepath.Join(parent, "myapp")
	expectFiles(t, appDir, []string{
		"cmd/server/main.go",
		"go.mod",
		".env.example",
		"Makefile",
		".gitignore",
	})
}

func TestRunWithFS_copiesExamplesToInternal(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appDir := filepath.Join(parent, "myapp")
	expectFiles(t, appDir, []string{
		"internal/clean_package_example_en/domain.go",
		"internal/clean_package_example_ru/domain.go",
	})
}

func TestRunWithFS_gitignoreIncludesExamples(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(parent, "myapp", ".gitignore"))
	if !strings.Contains(content, "clean_package_example") {
		t.Errorf(".gitignore should contain example patterns, got:\n%s", content)
	}
}

func TestRunWithFS_dirtyDir_returnsError(t *testing.T) {
	parent := t.TempDir()
	appDir := filepath.Join(parent, "myapp")
	os.MkdirAll(appDir, 0o755)
	writeFile(t, filepath.Join(appDir, "main.go"), "package main\n")

	cfg := initcmd.Config{AppName: "myapp", Dir: parent}
	err := initcmd.RunWithFS(cfg, fakeExamplesFS())
	if err == nil {
		t.Fatal("expected DirtyDirectoryError, got nil")
	}

	var dirtyErr *initcmd.DirtyDirectoryError
	if !errors.As(err, &dirtyErr) {
		t.Errorf("expected *DirtyDirectoryError, got %T: %v", err, err)
	}
	if len(dirtyErr.Files) == 0 {
		t.Error("DirtyDirectoryError should list the blocking files")
	}
}

func TestRunWithFS_dirtyDir_force_succeeds(t *testing.T) {
	parent := t.TempDir()
	appDir := filepath.Join(parent, "myapp")
	os.MkdirAll(appDir, 0o755)
	writeFile(t, filepath.Join(appDir, "main.go"), "package main\n")

	cfg := initcmd.Config{AppName: "myapp", Dir: parent, Force: true}
	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}
}

func TestRunWithFS_moduleFromGoMod(t *testing.T) {
	parent := t.TempDir()
	appDir := filepath.Join(parent, "myapp")
	os.MkdirAll(appDir, 0o755)
	writeFile(t, filepath.Join(appDir, "go.mod"), "module github.com/user/myapp\n\ngo 1.22\n")

	cfg := initcmd.Config{AppName: "myapp", Dir: parent}
	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(appDir, "go.mod"))
	if !strings.Contains(content, "github.com/user/myapp") {
		t.Errorf("expected module name from go.mod, got:\n%s", content)
	}
}

func TestRunWithFS_moduleFromFlag(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent, Module: "github.com/company/myapp"}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(parent, "myapp", "go.mod"))
	if !strings.Contains(content, "github.com/company/myapp") {
		t.Errorf("expected module from flag, got:\n%s", content)
	}
}

func TestRunWithFS_moduleFromAppName(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(parent, "myapp", "go.mod"))
	if !strings.Contains(content, "myapp") {
		t.Errorf("expected module name 'myapp', got:\n%s", content)
	}
}

func TestRunWithFS_createsGoMod_whenMissing(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(parent, "myapp", "go.mod")); err != nil {
		t.Errorf("expected go.mod to be created: %v", err)
	}
}

func TestRunWithFS_skipsGoMod_whenPresent(t *testing.T) {
	parent := t.TempDir()
	appDir := filepath.Join(parent, "myapp")
	os.MkdirAll(appDir, 0o755)

	original := "module github.com/user/myapp\n\ngo 1.22\n"
	writeFile(t, filepath.Join(appDir, "go.mod"), original)

	cfg := initcmd.Config{AppName: "myapp", Dir: parent}
	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(appDir, "go.mod"))
	if content != original {
		t.Errorf("go.mod should not be overwritten without --force, got:\n%s", content)
	}
}

func TestRunWithFS_mainGoContainsModuleName(t *testing.T) {
	parent := t.TempDir()
	cfg := initcmd.Config{AppName: "myapp", Dir: parent, Module: "github.com/acme/myapp"}

	if err := initcmd.RunWithFS(cfg, fakeExamplesFS()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := readFile(t, filepath.Join(parent, "myapp", "cmd", "server", "main.go"))
	if !strings.Contains(content, "github.com/acme/myapp") {
		t.Errorf("main.go should contain module name, got:\n%s", content)
	}
}

// --- helpers ---

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdirAll %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile %s: %v", path, err)
	}
	return string(b)
}

func expectFiles(t *testing.T, dir string, paths []string) {
	t.Helper()
	for _, p := range paths {
		full := filepath.Join(dir, p)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("expected file %s to exist: %v", p, err)
		}
	}
}
