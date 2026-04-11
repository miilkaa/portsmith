// Package initcmd implements the "portsmith init" command.
//
// The command scaffolds a new Clean Architecture application:
//
//	portsmith init <app-name> [--module <module-path>] [--force]
//
// Generated structure:
//
//	<app-name>/
//	├── cmd/server/main.go           ← manual dependency injection entry point
//	├── internal/
//	│   ├── clean_package_example_en/ ← reference examples (gitignored)
//	│   └── clean_package_example_ru/ ← reference examples (gitignored)
//	├── go.mod
//	├── .env.example
//	├── Makefile
//	└── .gitignore
package initcmd

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	portsmith "github.com/miilkaa/portsmith"
)

// Config holds all options for the init command.
type Config struct {
	AppName string // required positional argument
	Dir     string // base directory; app is created at Dir/AppName
	Module  string // Go module path; derived from AppName if empty
	Force   bool   // skip dirty-directory check
}

// DirtyDirectoryError is returned when the target directory already contains
// Go source files that indicate an existing application.
type DirtyDirectoryError struct {
	Files []string
}

func (e *DirtyDirectoryError) Error() string {
	listed := strings.Join(e.Files, "\n  ")
	return fmt.Sprintf(
		"target directory already contains Go source files:\n  %s\n\nUse --force to skip this check.",
		listed,
	)
}

// allowedTopLevel maps top-level names that do NOT indicate an existing Go app.
var allowedTopLevel = map[string]bool{
	"go.mod":          true,
	"go.sum":          true,
	".git":            true,
	".gitignore":      true,
	".gitattributes":  true,
	".env":            true,
	".env.example":    true,
	"Makefile":        true,
	".cursor":         true,
	".vscode":         true,
	".idea":           true,
	"docs":            true,
	"examples":        true,
	"LICENSE":         true,
	"CHANGELOG.md":    true,
}

// isAllowedEntry returns true for files/dirs that do not block init.
func isAllowedEntry(name string) bool {
	if allowedTopLevel[name] {
		return true
	}
	if strings.HasSuffix(name, ".md") {
		return true
	}
	if strings.HasPrefix(name, "README") {
		return true
	}
	return false
}

// CheckDirty scans dir and returns a list of .go files that would indicate
// an existing Go application. Non-Go and well-known meta files are ignored.
func CheckDirty(dir string) ([]string, error) {
	var blocking []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(dir, path)
		if rel == "." {
			return nil
		}

		// Determine the top-level entry name.
		top := strings.SplitN(rel, string(filepath.Separator), 2)[0]

		if isAllowedEntry(top) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			blocking = append(blocking, rel)
		}

		return nil
	})

	return blocking, err
}

// Run is the public entry point called by cmd/portsmith/main.go.
func Run(args []string) error {
	// Go's flag.FlagSet stops parsing at the first non-flag argument, so
	// "portsmith init myapp --module x" would not pick up --module.
	// Pre-process args to move all --flag [value] pairs before positionals.
	flagArgs, positionals := splitFlagsAndPositionals(args)
	sorted := append(flagArgs, positionals...)

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	var module string
	var force bool
	fs.StringVar(&module, "module", "", "Go module path (default: app-name)")
	fs.BoolVar(&force, "force", false, "skip dirty directory check")

	if err := fs.Parse(sorted); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("app-name is required\n\nusage: portsmith init <app-name> [--module <path>] [--force]")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	cfg := Config{
		AppName: remaining[0],
		Dir:     cwd,
		Module:  module,
		Force:   force,
	}

	return RunWithFS(cfg, portsmith.ExamplesFS)
}

// splitFlagsAndPositionals separates --flag [value] pairs from positional args
// so flags can appear in any position in the argument list.
func splitFlagsAndPositionals(args []string) (flags, positionals []string) {
	// boolFlagNames are flags that do not consume the following argument as value.
	boolFlagNames := map[string]bool{
		"--force": true, "-force": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}

		flags = append(flags, arg)

		// If the flag embeds its value (--module=foo), nothing to consume.
		if strings.Contains(arg, "=") {
			continue
		}

		// Bool flags never consume the next argument.
		if boolFlagNames[arg] {
			continue
		}

		// Consume the next token as the flag's value if it is not itself a flag.
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return
}

// RunWithFS is the testable implementation. It accepts an fs.FS so tests
// can inject a minimal fake instead of the full embedded examples.
func RunWithFS(cfg Config, examplesFS fs.FS) error {
	baseDir := cfg.Dir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}

	appDir := filepath.Join(baseDir, cfg.AppName)

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return fmt.Errorf("creating app directory: %w", err)
	}

	// Dirty-directory check.
	if !cfg.Force {
		blocking, err := CheckDirty(appDir)
		if err != nil {
			return fmt.Errorf("checking directory: %w", err)
		}
		if len(blocking) > 0 {
			return &DirtyDirectoryError{Files: blocking}
		}
	}

	// Resolve module name.
	module := cfg.Module
	if module == "" {
		var err error
		module, err = readModuleFromGoMod(filepath.Join(appDir, "go.mod"))
		if err != nil {
			// Fallback: use app name as module name.
			module = cfg.AppName
		}
	}

	data := templateData{
		AppName: cfg.AppName,
		Module:  module,
		GoVer:   goVersion(),
	}

	// Write scaffold files (skip existing unless --force).
	if err := writeScaffold(appDir, data, cfg.Force); err != nil {
		return err
	}

	// Copy embedded examples into internal/.
	if err := copyExamples(appDir, examplesFS); err != nil {
		return fmt.Errorf("copying examples: %w", err)
	}

	printNextSteps(cfg.AppName, module)
	return nil
}

// --- template data ---

type templateData struct {
	AppName string
	Module  string
	GoVer   string // e.g. "1.22"
}

// --- scaffold files ---

func writeScaffold(appDir string, data templateData, force bool) error {
	files := []struct {
		relPath  string
		content  string
		skipIfEx bool // do not overwrite if already exists (unless force)
	}{
		{relPath: "cmd/server/main.go", content: mainGoTpl, skipIfEx: !force},
		{relPath: "go.mod", content: goModTpl, skipIfEx: true}, // never overwrite go.mod by default
		{relPath: ".env.example", content: envExampleTpl, skipIfEx: !force},
		{relPath: "Makefile", content: makefileTpl, skipIfEx: !force},
		{relPath: ".gitignore", content: gitignoreTpl, skipIfEx: !force},
	}

	for _, f := range files {
		fullPath := filepath.Join(appDir, filepath.FromSlash(f.relPath))

		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("mkdir for %s: %w", f.relPath, err)
		}

		if f.skipIfEx {
			if _, err := os.Stat(fullPath); err == nil {
				fmt.Printf("  skip   %s\n", f.relPath)
				continue
			}
		}

		rendered, err := renderTemplate(f.content, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", f.relPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(rendered), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f.relPath, err)
		}

		fmt.Printf("  create %s\n", f.relPath)
	}

	return nil
}

// copyExamples extracts "examples/*" from examplesFS into <appDir>/internal/.
// Existing files are silently skipped.
func copyExamples(appDir string, examplesFS fs.FS) error {
	return fs.WalkDir(examplesFS, "examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "examples" {
			return nil
		}

		// Strip "examples/" prefix; put files under internal/.
		rel := strings.TrimPrefix(path, "examples/")
		dest := filepath.Join(appDir, "internal", filepath.FromSlash(rel))

		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		// Skip if already exists.
		if _, err := os.Stat(dest); err == nil {
			return nil
		}

		src, err := examplesFS.Open(path)
		if err != nil {
			return fmt.Errorf("opening embedded %s: %w", path, err)
		}
		defer src.Close()

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		dst, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("creating %s: %w", dest, err)
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		return err
	})
}

// --- helpers ---

func renderTemplate(tplContent string, data templateData) (string, error) {
	tpl, err := template.New("").Parse(tplContent)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// readModuleFromGoMod reads the module directive from go.mod.
func readModuleFromGoMod(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// goVersion returns the major.minor version from runtime.Version().
// e.g. "go1.22.5" → "1.22"
func goVersion() string {
	v := strings.TrimPrefix(runtime.Version(), "go")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

func printNextSteps(appName, module string) {
	fmt.Printf(`
  Project "%s" initialised successfully.

  Module: %s

  Next steps:
    cd %s
    go mod tidy
    portsmith new internal/<package>
    portsmith gen  internal/<package>
    portsmith mock internal/<package>
    go run ./cmd/server
`, appName, module, appName)
}

// --- templates ---

const mainGoTpl = `package main

import (
	"log"

	"github.com/miilkaa/portsmith/pkg/config"
	"github.com/miilkaa/portsmith/pkg/database"
	"github.com/miilkaa/portsmith/pkg/server"

	// Uncomment after creating your first package:
	// "{{.Module}}/internal/<package>"
)

// Config holds all application settings loaded from environment variables.
type Config struct {
	Port        int    ` + "`" + `env:"PORT"         env-default:"8080"` + "`" + `
	DatabaseURL string ` + "`" + `env:"DATABASE_URL" env-required:"true"` + "`" + `
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(database.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	// Register domain models for AutoMigrate:
	// database.Register(db, &orders.Order{})

	srv := server.New(server.Config{Port: cfg.Port})

	// Wire your handlers here:
	// repo := orders.NewRepository(db.DB())
	// svc  := orders.NewService(repo)
	// h    := orders.NewHandler(svc)
	// h.Routes(srv.Router().Group("/api/v1"))

	_ = db // remove after wiring
	log.Fatal(srv.Run())
}
`

const goModTpl = `module {{.Module}}

go {{.GoVer}}

require github.com/miilkaa/portsmith v0.1.0
`

const envExampleTpl = `PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/{{.AppName}}?sslmode=disable
`

const makefileTpl = `.PHONY: run test gen mock check

run:
	go run ./cmd/server

test:
	go test ./...

gen:
	portsmith gen ./internal/...

mock:
	portsmith mock ./internal/...

check:
	portsmith check ./internal/...
`

const gitignoreTpl = `# Binaries
bin/
*.exe
*.dll
*.so
*.dylib

# Test output
*.test
*.out

# Environment
.env

# portsmith reference examples (read-only, not part of your code)
internal/clean_package_example_en/
internal/clean_package_example_ru/
`
