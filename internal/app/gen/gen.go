// Package gen implements the portsmith gen command.
package gen

import (
	"fmt"
	"time"

	"github.com/miilkaa/portsmith/internal/analyze"
	"github.com/miilkaa/portsmith/internal/workpool"
	"golang.org/x/tools/go/packages"
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

	progress := newProgressLogger(opts.verbose)
	started := time.Now()
	progress.printf("portsmith gen: workers=%d packages=%d\n", workpool.WorkerCount(len(dirs)), len(dirs))
	defer func() {
		progress.printf("portsmith gen: completed in %s\n", time.Since(started).Round(time.Millisecond))
	}()

	configs, err := checkCallPatternsBeforeGen(dirs, progress)
	if err != nil {
		return err
	}

	// When --scan-callers is enabled, load every package of the current module
	// once with full type information and reuse for every genPackage call. This
	// is the slow path (go/packages does full type-checking) but it is precise.
	var (
		modulePath string
		modulePkgs []*packages.Package
	)
	if opts.scanCallers {
		mp, err := analyze.DetectModulePath(".")
		if err != nil {
			return fmt.Errorf("--scan-callers requires a go.mod in the current dir: %w", err)
		}
		modulePath = mp
		modulePkgs, err = analyze.LoadModulePackages(".")
		if err != nil {
			return fmt.Errorf("load module packages: %w", err)
		}
	}

	genResults := workpool.Run(dirs, func(_ int, dir string) (string, error) {
		progress.packageStart("generate", dir)
		started := time.Now()
		out, err := genPackage(dir, configs[dir], opts.dryRun, opts.scanCallers, modulePath, modulePkgs)
		progress.packageDone("generate", dir, started, err)
		return out, err
	})
	for _, result := range genResults {
		if result.Err != nil {
			return fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		if opts.dryRun && result.Value != "" {
			fmt.Print(result.Value)
		}
	}
	return nil
}
