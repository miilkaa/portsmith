// Package check implements the portsmith check command — CLI entry for the architecture linter.
//
// Usage in CI/CD:
//
//	portsmith check ./internal/...
//
// Exit code 1 is returned when error-severity violations are found.
// Warning-severity violations are printed but do not fail the command.
package check

import (
	"fmt"

	"github.com/miilkaa/portsmith/internal/workpool"
)

// Run executes the check command for the given arguments.
func Run(args []string) error {
	var report checkReport

	checkOpts := parseArgs(args)
	if len(checkOpts.patterns) == 0 {
		checkOpts.patterns = []string{"./..."}
	}

	dirs, err := resolveDirs(checkOpts.patterns)
	if err != nil {
		return err
	}

	results := workpool.Run(dirs, func(_ int, dir string) (packageResult, error) {
		return checkPackage(dir)
	})

	for _, result := range results {
		if result.Err != nil {
			return fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		report.addPackage(result.Value)
	}

	printReport(report)
	if report.hasErrors() {
		return fmt.Errorf("%d error violation(s)", report.errorCount())
	}

	return nil
}
