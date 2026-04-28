// Command portsmith is the CLI entry point for the portsmith framework tools.
//
// Available commands:
//
//	portsmith init    [--force]                                   — interactive portsmith.yaml wizard
//	portsmith gen     [--dry-run] [--all] [<pkg-dir>...]        — generate ports.go
//	portsmith new     <pkg-dir>                                 — scaffold a new package
//	portsmith mock    [<pkg-dir>...]                            — generate mocks via mockery
//	portsmith check   [<pkg-dir>...]                            — architecture linter
//	portsmith version                                           — print version
package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/miilkaa/portsmith/cmd/portsmith/check"
	gencmd "github.com/miilkaa/portsmith/cmd/portsmith/gen"
	initcmd "github.com/miilkaa/portsmith/cmd/portsmith/init"
	mockcmd "github.com/miilkaa/portsmith/cmd/portsmith/mock"
	newcmd "github.com/miilkaa/portsmith/cmd/portsmith/new"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = initcmd.Run(args)
	case "gen":
		err = gencmd.Run(args)
	case "new":
		err = newcmd.Run(args)
	case "mock":
		err = mockcmd.Run(args)
	case "check":
		err = check.Run(args)
	case "version", "--version", "-v":
		fmt.Println(buildVersion())
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "portsmith: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "portsmith %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

// buildVersion returns the module version embedded by the Go toolchain at build time.
// Returns "(devel)" when running via `go run` or an untagged local build.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return "portsmith " + info.Main.Version
	}
	// For local builds, show VCS commit if available.
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			dirty := ""
			for _, ss := range info.Settings {
				if ss.Key == "vcs.modified" && ss.Value == "true" {
					dirty = "-dirty"
					break
				}
			}
			return "portsmith (devel) " + s.Value[:7] + dirty
		}
	}
	return "portsmith (devel)"
}

func printUsage() {
	fmt.Print(`portsmith — Go Clean Architecture toolkit

Usage:
  portsmith init  [--force]
      Interactive wizard: writes portsmith.yaml in the current directory.
      Prompt language follows LC_ALL / LC_MESSAGES / LANG (Russian if locale
      starts with "ru", otherwise English). If go.mod exists, its module path
      is added as a comment in the generated file.
      --force   overwrite an existing portsmith.yaml

  portsmith gen   [--dry-run] [--all] [<pkg-dir>...]
      Generate ports.go for one or more packages.
      --all     scan all packages under internal/
      --dry-run print generated content without writing files

  portsmith new   [--stack gin-gorm|chi-sqlx] <pkg-dir>
      Scaffold a new package with domain/service/repository/handler/dto files.
      Stack defaults from portsmith.yaml or go.mod (Chi → chi-sqlx, Gin → gin-gorm).

  portsmith mock  [<pkg-dir>...]
      Generate mocks for all interfaces in ports.go via mockery.

  portsmith check [--stack gin-gorm|chi-sqlx] [<pkg-dir>...]
      Validate Clean Architecture rules. Exits with code 1 on violations.
      Prints detected stack; override with --stack. Suitable for CI/CD pipelines.

  portsmith version
      Print the installed version.

  portsmith help
      Print this help message.

Examples:
  portsmith init
  portsmith init --force
  portsmith gen --all
  portsmith gen internal/orders
  portsmith new internal/products
  portsmith new --stack chi-sqlx internal/widgets
  portsmith mock internal/orders internal/products
  portsmith check ./internal/...
`)
}
