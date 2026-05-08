package check

import (
	"reflect"
	"testing"
)

func TestParseArgs_collectsPatterns(t *testing.T) {
	opts := parseArgs([]string{"./...", "internal/orders"})

	want := []string{"./...", "internal/orders"}
	if !reflect.DeepEqual(opts.patterns, want) {
		t.Fatalf("patterns = %v, want %v", opts.patterns, want)
	}
}

func TestParseArgs_emptyArgs(t *testing.T) {
	opts := parseArgs(nil)

	if len(opts.patterns) != 0 {
		t.Fatalf("patterns = %v, want empty", opts.patterns)
	}
}
