package check

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/miilkaa/portsmith/internal/lint"
)

func TestCheckReport_addPackageAndCounts(t *testing.T) {
	var report checkReport

	report.addPackage(packageResult{
		errViolations: []lint.Violation{
			{Rule: "handler-no-db"},
		},
		warnViolations: []lint.Violation{
			{Rule: "service-no-http"},
		},
	})

	if !report.hasViolations() {
		t.Fatal("expected report to have violations")
	}
	if !report.hasErrors() {
		t.Fatal("expected report to have errors")
	}
	if got := report.errorCount(); got != 1 {
		t.Fatalf("errorCount = %d, want 1", got)
	}
	if len(report.warns) != 1 {
		t.Fatalf("warn count = %d, want 1", len(report.warns))
	}
}

func TestViolationSummary_sortsRuleCounts(t *testing.T) {
	errs := []lint.Violation{
		{Rule: "service-no-http"},
		{Rule: "handler-no-db"},
	}
	warns := []lint.Violation{
		{Rule: "handler-no-db"},
	}

	got := violationSummary(errs, warns)
	want := "3 violation(s) — handler-no-db: 2, service-no-http: 1\n"
	if got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}

func TestPrintReport_printsWarningsBeforeErrorsAndSummary(t *testing.T) {
	report := checkReport{
		warns: []lint.Violation{
			{Rule: "z-warning", File: "b.go", Line: 2, Message: "warn"},
		},
		errs: []lint.Violation{
			{Rule: "a-error", File: "a.go", Line: 1, Message: "err"},
		},
	}

	out, err := captureCheckStdout(func() error {
		printReport(report)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	warnIdx := strings.Index(out, "warning [z-warning]")
	errIdx := strings.Index(out, "error   [a-error]")
	if warnIdx < 0 || errIdx < 0 {
		t.Fatalf("expected warning and error lines, got:\n%s", out)
	}
	if warnIdx > errIdx {
		t.Fatalf("warnings should print before errors, got:\n%s", out)
	}
	if !strings.Contains(out, "2 violation(s) — a-error: 1, z-warning: 1") {
		t.Fatalf("expected summary, got:\n%s", out)
	}
}

func TestPrintReport_emptyReportPrintsNothing(t *testing.T) {
	out, err := captureCheckStdout(func() error {
		printReport(checkReport{})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("output = %q, want empty", out)
	}
}

func captureCheckStdout(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	runErr := fn()

	closeErr := w.Close()
	if closeErr != nil {
		return "", closeErr
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	if err := r.Close(); err != nil {
		return "", err
	}
	return buf.String(), runErr
}
