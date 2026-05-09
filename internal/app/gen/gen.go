// Package gen implements the portsmith gen command.
package gen

import (
	"fmt"
)

// Run executes the gen command with the given arguments.
func Run(args []string) error {
	opts := parseArgs(args)

	dirs, err := resolveGenDirs(opts.patterns)
	if err != nil {
		return err
	}

	if len(dirs) == 0 {
		return fmt.Errorf("no package directories specified")
	}

	progress, finishProgress := startProgress(opts.verbose, len(dirs))
	defer finishProgress()

	configs, err := checkCallPatternsBeforeGen(dirs, progress)
	if err != nil {
		return err
	}

	callerCtx, err := loadCallerScanContext(opts.scanCallers)
	if err != nil {
		return err
	}

	return generatePackages(dirs, configs, opts, callerCtx, progress)
}
