package check

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
		if isRecursivePattern(pattern) {
			root := recursiveRoot(pattern)

			err := walkDirs(root, addDir)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}

		addDir(pattern)
	}
	return result, nil
}

func isRecursivePattern(pattern string) bool {
	return strings.HasSuffix(pattern, "/...")
}

func recursiveRoot(pattern string) string {
	root := strings.TrimSuffix(pattern, "/...")
	root = strings.TrimPrefix(root, "./")
	if root == "" {
		return "."
	}
	return root
}

func shouldSkipDir(name string) bool {
	return name == "vendor" || name == ".git"
}

func walkDirs(root string, add func(string)) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		add(path)
		return nil
	})
}
