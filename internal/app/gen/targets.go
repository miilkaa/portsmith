package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miilkaa/portsmith/internal/app/target"
)

// Target resolution.

func resolveGenDirs(ptrns []string) ([]string, error) {
	targets := newGenTargets()

	for _, pattern := range ptrns {
		if err := addGenTargets(pattern, targets); err != nil {
			return nil, err
		}
	}
	return targets.values(), nil
}

func addGenTargets(pattern string, targets *genTargets) error {
	switch {
	case isGenRecursivePattern(pattern):
		return addRecursiveGenTargets(pattern, targets)
	case hasGlob(pattern):
		return addGlobGenTargets(pattern, targets)
	case isExistingDir(pattern):
		targets.addPackage(pattern)
	default:
		targets.addPath(pattern)
	}
	return nil
}

// Recursive patterns.

func addRecursiveGenTargets(pattern string, targets *genTargets) error {
	return walkGenPackages(recursiveGenRoot(pattern), targets)
}

func walkGenPackages(root string, targets *genTargets) error {
	return target.WalkDirs(root, func(dir string) {
		targets.addPackage(dir)
	})
}

func isGenRecursivePattern(pattern string) bool {
	return pattern == "..." || target.IsRecursivePattern(pattern)
}

func recursiveGenRoot(pattern string) string {
	if pattern == "..." {
		return "."
	}
	return target.RecursiveRoot(pattern)
}

// Glob patterns.

func addGlobGenTargets(pattern string, targets *genTargets) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("pattern %q matched no package directories", pattern)
	}

	matchedPackage := false
	for _, match := range matches {
		if targets.addPackage(match) {
			matchedPackage = true
		}
	}
	if !matchedPackage {
		return fmt.Errorf("pattern %q matched no package directories", pattern)
	}
	return nil
}

func hasGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// Plain paths.

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Target collection.

type genTargets struct {
	seen map[string]bool
	dirs []string
}

func newGenTargets() *genTargets {
	return &genTargets{seen: make(map[string]bool)}
}

func (t *genTargets) addPath(path string) {
	path = filepath.Clean(path)
	if t.seen[path] {
		return
	}
	t.seen[path] = true
	t.dirs = append(t.dirs, path)
}

func (t *genTargets) addPackage(dir string) bool {
	if !isGenPackage(dir) {
		return false
	}
	t.addPath(dir)
	return true
}

func (t *genTargets) values() []string {
	return t.dirs
}

// Package detection.

var requiredGenPackageFiles = []string{
	"handler.go",
	"service.go",
	"repository.go",
}

func isGenPackage(dir string) bool {
	return hasFiles(dir, requiredGenPackageFiles...)
}

func hasFiles(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}
