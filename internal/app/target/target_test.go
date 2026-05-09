package target

import "testing"

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
			if got := IsRecursivePattern(tt.pattern); got != tt.recursive {
				t.Fatalf("IsRecursivePattern(%q) = %v, want %v", tt.pattern, got, tt.recursive)
			}
			if !tt.recursive {
				return
			}
			if got := RecursiveRoot(tt.pattern); got != tt.root {
				t.Fatalf("RecursiveRoot(%q) = %q, want %q", tt.pattern, got, tt.root)
			}
		})
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
			if got := ShouldSkipDir(tt.name); got != tt.want {
				t.Fatalf("ShouldSkipDir(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
