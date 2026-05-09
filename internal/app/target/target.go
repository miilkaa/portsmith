package target

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func IsRecursivePattern(pattern string) bool {
	return strings.HasSuffix(pattern, "/...")
}

func RecursiveRoot(pattern string) string {
	root := strings.TrimSuffix(pattern, "/...")
	root = strings.TrimPrefix(root, "./")
	if root == "" {
		return "."
	}
	return root
}

func ShouldSkipDir(name string) bool {
	return name == "vendor" || name == ".git"
}

func WalkDirs(root string, add func(string)) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		if ShouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		add(path)
		return nil
	})
}
