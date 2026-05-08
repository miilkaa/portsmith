package check

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecursivePattern(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		recursive bool
		root      string
	}{
		{
			name:      "current directory",
			pattern:   "./...",
			recursive: true,
			root:      ".",
		},
		{
			name:      "nested relative directory",
			pattern:   "./internal/...",
			recursive: true,
			root:      "internal",
		},
		{
			name:      "plain nested directory",
			pattern:   "internal/...",
			recursive: true,
			root:      "internal",
		},
		{
			name:      "plain directory",
			pattern:   "internal",
			recursive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRecursivePattern(tt.pattern); got != tt.recursive {
				t.Fatalf("isRecursivePattern(%q) = %v, want %v", tt.pattern, got, tt.recursive)
			}
			if !tt.recursive {
				return
			}
			if got := recursiveRoot(tt.pattern); got != tt.root {
				t.Fatalf("recursiveRoot(%q) = %q, want %q", tt.pattern, got, tt.root)
			}
		})
	}
}

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

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "vendor", want: true},
		{name: ".git", want: true},
		{name: "notvendor", want: false},
		{name: "git", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipDir(tt.name); got != tt.want {
				t.Fatalf("shouldSkipDir(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func writeTargetDir(t *testing.T, parts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(parts...), 0o755); err != nil {
		t.Fatal(err)
	}
}
