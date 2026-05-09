package gen

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveGenDirs_recursivePatternFindsOnlyGenPackages(t *testing.T) {
	root := t.TempDir()
	writePackage(t, filepath.Join(root, "app", "orders"))
	writePackage(t, filepath.Join(root, "app", "nested", "payments"))
	writeFile(t, filepath.Join(root, "app", "shared"), "helper.go", "package shared\n")

	dirs, err := resolveGenDirs([]string{filepath.Join(root, "app") + "/..."})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "app", "nested", "payments"),
		filepath.Join(root, "app", "orders"),
	}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func TestResolveGenDirs_ellipsisFindsOnlyGenPackages(t *testing.T) {
	root := t.TempDir()
	writePackage(t, filepath.Join(root, "orders"))
	writeFile(t, filepath.Join(root, "shared"), "helper.go", "package shared\n")

	withWorkingDir(t, root, func() {
		dirs, err := resolveGenDirs([]string{"..."})
		if err != nil {
			t.Fatal(err)
		}

		want := []string{"orders"}
		if !reflect.DeepEqual(dirs, want) {
			t.Fatalf("dirs = %v, want %v", dirs, want)
		}
	})
}

func TestResolveGenDirs_globFindsOnlyGenPackages(t *testing.T) {
	root := t.TempDir()
	writePackage(t, filepath.Join(root, "internal", "orders"))
	writePackage(t, filepath.Join(root, "internal", "payments"))
	writeFile(t, filepath.Join(root, "internal", "shared"), "helper.go", "package shared\n")

	withWorkingDir(t, root, func() {
		dirs, err := resolveGenDirs([]string{"internal/*"})
		if err != nil {
			t.Fatal(err)
		}

		want := []string{
			filepath.Join("internal", "orders"),
			filepath.Join("internal", "payments"),
		}
		if !reflect.DeepEqual(dirs, want) {
			t.Fatalf("dirs = %v, want %v", dirs, want)
		}
	})
}

func TestResolveGenDirs_plainDirSkipsNonGenPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "shared"), "helper.go", "package shared\n")

	dirs, err := resolveGenDirs([]string{filepath.Join(root, "shared")})
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Fatalf("dirs = %v, want empty", dirs)
	}
}

func TestResolveGenDirs_missingPlainPathIsReturned(t *testing.T) {
	dirs, err := resolveGenDirs([]string{"internal/missing"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"internal/missing"}
	if !reflect.DeepEqual(dirs, want) {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore wd: %v", err)
		}
	}()

	fn()
}
