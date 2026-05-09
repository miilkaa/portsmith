package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miilkaa/portsmith/internal/app/target"
)

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
		if !isGenPackage(dir) {
			return false
		}
		add(dir)
		return true
	}

	walkGenPackages := func(root string) error {
		return target.WalkDirs(root, func(dir string) {
			addIfGenPackage(dir)
		})
	}

	addGlobMatches := func(pattern string) error {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("pattern %q matched no package directories", pattern)
		}

		matchedPackage := false
		for _, match := range matches {
			if addIfGenPackage(match) {
				matchedPackage = true
			}
		}
		if !matchedPackage {
			return fmt.Errorf("pattern %q matched no package directories", pattern)
		}
		return nil
	}

	for _, p := range ptrns {
		if p == "..." || target.IsRecursivePattern(p) {
			root := "."
			if p != "..." {
				root = target.RecursiveRoot(p)
			}
			if err := walkGenPackages(root); err != nil {
				return nil, err
			}
			continue
		}

		if hasGlob(p) {
			if err := addGlobMatches(p); err != nil {
				return nil, err
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

func isGenPackage(dir string) bool {
	return hasFiles(dir, "handler.go", "service.go", "repository.go")
}
