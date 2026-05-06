// Package cli handles command selection and terminal-facing behavior.
package cli

import (
	"fmt"

	"github.com/miilkaa/portsmith/internal/app/check"
	"github.com/miilkaa/portsmith/internal/app/gen"
	"github.com/miilkaa/portsmith/internal/app/initconfig"
	"github.com/miilkaa/portsmith/internal/app/mock"
	"github.com/miilkaa/portsmith/internal/app/scaffold"
)

// Run dispatches portsmith command-line arguments.
func Run(args []string) error {
	if len(args) < 1 {
		PrintUsage()
		return fmt.Errorf("missing command")
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "init":
		return initconfig.Run(rest)
	case "gen":
		return gen.Run(rest)
	case "new":
		return scaffold.Run(rest)
	case "mock":
		return mock.Run(rest)
	case "check":
		return check.Run(rest)
	case "version", "--version", "-v":
		fmt.Println(buildVersion())
		return nil
	case "help", "--help", "-h":
		PrintUsage()
		return nil
	default:
		PrintUsage()
		return fmt.Errorf("portsmith: unknown command %q", cmd)
	}
}
