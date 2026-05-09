package check

import (
	"os"
	"path/filepath"

	"github.com/miilkaa/portsmith/internal/app/target"
)

// resolveDirs expands Go-style patterns like ./internal/... into directory paths.
func resolveDirs(patterns []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string
	addDir := func(dir string) {
		dir = filepath.Clean(dir)
		if !seen[dir] {
			seen[dir] = true
			result = append(result, dir)
		}
	}

	for _, pattern := range patterns {
		if target.IsRecursivePattern(pattern) {
			root := target.RecursiveRoot(pattern)

			err := target.WalkDirs(root, addDir)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}

		addDir(pattern)
	}
	return result, nil
}
