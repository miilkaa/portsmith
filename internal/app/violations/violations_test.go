package violations

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/miilkaa/portsmith/internal/lint"
)

func TestSummary_sortsRuleCounts(t *testing.T) {
	errs := []lint.Violation{
		{Rule: "service-no-http"},
		{Rule: "handler-no-db"},
	}
	warns := []lint.Violation{
		{Rule: "handler-no-db"},
	}

	got := Summary(errs, warns)
	want := "3 violation(s) — handler-no-db: 2, service-no-http: 1\n"
	if got != want {
		t.Fatalf("Summary = %q, want %q", got, want)
	}
}

func TestPrintLines_printsWarningsBeforeErrors(t *testing.T) {
	warns := []lint.Violation{
		{Rule: "z-warning", File: "b.go", Line: 2, Message: "warn"},
	}
	errs := []lint.Violation{
		{Rule: "a-error", File: "a.go", Line: 1, Message: "err"},
	}

	out := captureStdout(t, func() {
		PrintLines(warns, errs)
	})

	warnIdx := strings.Index(out, "warning [z-warning]")
	errIdx := strings.Index(out, "error   [a-error]")
	if warnIdx < 0 || errIdx < 0 {
		t.Fatalf("expected warning and error lines, got:\n%s", out)
	}
	if warnIdx > errIdx {
		t.Fatalf("warnings should print before errors, got:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
