package check

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveDirs_deduplicatesAndCleansPlainPatterns(t *testing.T) {
	dirs, err := resolveDirs([]string{"./internal/orders", "internal/orders", "internal/payments"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"internal/orders", "internal/payments"}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func TestResolveDirs_recursiveSkipsVendorAndGitDirs(t *testing.T) {
	root := t.TempDir()
	writeTargetDir(t, root, "app")
	writeTargetDir(t, root, "app", "orders")
	writeTargetDir(t, root, "app", "vendor")
	writeTargetDir(t, root, "app", "vendor", "dep")
	writeTargetDir(t, root, "app", ".git")
	writeTargetDir(t, root, "app", ".git", "objects")
	writeTargetDir(t, root, "app", "notvendor")

	dirs, err := resolveDirs([]string{filepath.Join(root, "app") + "/..."})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "app"),
		filepath.Join(root, "app", "notvendor"),
		filepath.Join(root, "app", "orders"),
	}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func writeTargetDir(t *testing.T, parts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(parts...), 0o755); err != nil {
		t.Fatal(err)
	}
}
