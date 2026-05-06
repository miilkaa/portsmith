// Package gen implements the portsmith gen command.
package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/miilkaa/portsmith/internal/analyze"
	"github.com/miilkaa/portsmith/internal/workpool"
	"golang.org/x/tools/go/packages"
)

// Run executes the gen command with the given arguments.
func Run(args []string) error {
	dryRun := false
	scanCallers := false
	verbose := false
	var dirs []string

	for _, a := range args {
		switch a {
		case "--dry-run":
			dryRun = true
		case "--scan-callers":
			scanCallers = true
		case "-v", "--verbose":
			verbose = true
		default:
			dirs = append(dirs, a)
		}
	}

	dirs, err := resolveGenDirs(dirs)
	if err != nil {
		return err
	}

	if len(dirs) == 0 {
		return fmt.Errorf("no package directories specified")
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

func resolveGenDirs(ptrns []string) ([]string, error) {
	seen := make(map[string]bool)
	var res []string

	add := func(dir string) {
		dir = filepath.Clean(dir)
		if !seen[dir] {
			seen[dir] = true
			res = append(res, dir)
		}
	}

	addIfGenPackage := func(dir string) bool {
		if hasFiles(dir, "handler.go", "service.go", "repository.go") {
			add(dir)
			return true
		}
		return false
	}

	walkGenPackages := func(root string) error {
		root = filepath.Clean(root)
		return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return err
			}
			switch filepath.Base(path) {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			addIfGenPackage(path)
			return nil
		})
	}

	for _, p := range ptrns {
		if p == "./..." || p == "..." {
			if err := walkGenPackages("."); err != nil {
				return nil, err
			}
			continue
		}

		if strings.HasSuffix(p, "/...") {
			root := strings.TrimSuffix(p, "/...")
			if root == "" {
				root = "."
			}
			if err := walkGenPackages(root); err != nil {
				return nil, err
			}
			continue
		}

		if hasGlob(p) {
			matches, err := filepath.Glob(p)
			if err != nil {
				return nil, err
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("pattern %q matched no package directories", p)
			}
			matchedPackage := false
			for _, m := range matches {
				if addIfGenPackage(m) {
					matchedPackage = true
				}
			}
			if !matchedPackage {
				return nil, fmt.Errorf("pattern %q matched no package directories", p)
			}
			continue
		}

		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			addIfGenPackage(p)
			continue
		}
		add(p)
	}
	return res, nil
}

func hasGlob(ptrn string) bool {
	return strings.ContainsAny(ptrn, "*?[")
}

func hasFiles(dir string, names ...string) bool {
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			return false
		}
	}
	return true
}
