// Package gen implements the portsmith gen command.
package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/miilkaa/portsmith/internal/analyze"
	"github.com/miilkaa/portsmith/internal/workpool"
	"golang.org/x/tools/go/packages"
)

// Run executes the gen command with the given arguments.
func Run(args []string) error {
	dryRun := false
	all := false
	scanCallers := false
	verbose := false
	var dirs []string

	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--all":
			all = true
		case "--scan-callers":
			scanCallers = true
		case "-v", "--verbose":
			verbose = true
		default:
			dirs = append(dirs, a)
		}
	}

	if all {
		matches, _ := filepath.Glob("internal/*")
		for _, m := range matches {
			fi, err := os.Stat(m)
			if err != nil || !fi.IsDir() {
				continue
			}
			if hasFiles(m, "handler.go", "service.go", "repository.go") {
				dirs = append(dirs, m)
			}
		}
		sort.Strings(dirs)
	}

	if len(dirs) == 0 {
		return fmt.Errorf("no package directories specified (use --all or provide paths)")
	}

	progress := newProgressLogger(verbose)
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
	if scanCallers {
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
		out, err := genPackage(dir, configs[dir], dryRun, scanCallers, modulePath, modulePkgs)
		progress.packageDone("generate", dir, started, err)
		return out, err
	})
	for _, result := range genResults {
		if result.Err != nil {
			return fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		if dryRun && result.Value != "" {
			fmt.Print(result.Value)
		}
	}
	return nil
}
func hasFiles(dir string, names ...string) bool {
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			return false
		}
	}
	return true
}
