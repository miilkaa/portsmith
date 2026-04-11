// Command portsmith is the CLI entry point for the portsmith framework tools.
//
// Available commands:
//
//	portsmith gen   [--dry-run] [--all] [<pkg-dir>...]  — generate ports.go
//	portsmith new   <pkg-dir>                            — scaffold a new package
//	portsmith mock  [<pkg-dir>...]                       — generate mocks via mockery
//	portsmith check [<pkg-dir>...]                       — architecture linter
package main

import (
	"fmt"
	"os"

	"github.com/miilkaa/portsmith/cmd/portsmith/check"
	gencmd "github.com/miilkaa/portsmith/cmd/portsmith/gen"
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
	case "gen":
		err = gencmd.Run(args)
	case "new":
		err = newcmd.Run(args)
	case "mock":
		err = mockcmd.Run(args)
	case "check":
		err = check.Run(args)
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

func printUsage() {
	fmt.Print(`portsmith — Go Clean Architecture toolkit

Usage:
  portsmith gen   [--dry-run] [--all] [<pkg-dir>...]
      Generate ports.go for one or more packages.
      --all     scan all packages under internal/
      --dry-run print generated content without writing files

  portsmith new   <pkg-dir>
      Scaffold a new package with domain/service/repository/handler/dto files.

  portsmith mock  [<pkg-dir>...]
      Generate mocks for all interfaces in ports.go via mockery.

  portsmith check [<pkg-dir>...]
      Validate Clean Architecture rules. Exits with code 1 on violations.
      Suitable for CI/CD pipelines.

  portsmith help
      Print this help message.

Examples:
  portsmith gen --all
  portsmith gen internal/orders
  portsmith new internal/products
  portsmith mock internal/orders internal/products
  portsmith check ./internal/...
`)
}
