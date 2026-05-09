package gen

import (
	"strings"
	"testing"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/project"
)

func TestCallPatternReport_addPackageAndCounts(t *testing.T) {
	report := newCallPatternReport(1)

	report.addPackage("internal/orders", callPatternResult{
		cfg: project.Config{},
		errViolations: []lint.Violation{
			{Rule: "call-pattern"},
		},
		warnViolations: []lint.Violation{
			{Rule: "call-pattern"},
		},
	})

	if !report.hasErrors() {
		t.Fatal("expected report to have errors")
	}
	if got := report.errorCount(); got != 1 {
		t.Fatalf("errorCount = %d, want 1", got)
	}
	if _, ok := report.configs["internal/orders"]; !ok {
		t.Fatal("expected config to be stored by package dir")
	}
	if len(report.warns) != 1 {
		t.Fatalf("warn count = %d, want 1", len(report.warns))
	}
}

func TestCallPatternResult_addViolationRespectsSeverity(t *testing.T) {
	errorCfg := project.Config{
		Lint: project.LintConfig{
			Rules: map[string]project.RuleConfig{
				"call-pattern": {Severity: "error"},
			},
		},
	}
	warningCfg := project.Config{
		Lint: project.LintConfig{
			Rules: map[string]project.RuleConfig{
				"call-pattern": {Severity: "warning"},
			},
		},
	}
	offCfg := project.Config{
		Lint: project.LintConfig{
			Rules: map[string]project.RuleConfig{
				"call-pattern": {Severity: "off"},
			},
		},
	}

	var result callPatternResult
	result.addViolation(errorCfg, lint.Violation{Rule: "call-pattern"})
	result.addViolation(warningCfg, lint.Violation{Rule: "call-pattern"})
	result.addViolation(offCfg, lint.Violation{Rule: "call-pattern"})

	if len(result.errViolations) != 1 {
		t.Fatalf("error count = %d, want 1", len(result.errViolations))
	}
	if len(result.warnViolations) != 1 {
		t.Fatalf("warning count = %d, want 1", len(result.warnViolations))
	}
}

func TestPrintCallPatternReport_printsWarningsBeforeErrors(t *testing.T) {
	report := callPatternReport{
		warns: []lint.Violation{
			{Rule: "z-warning", File: "b.go", Line: 2, Message: "warn"},
		},
		errs: []lint.Violation{
			{Rule: "a-error", File: "a.go", Line: 1, Message: "err"},
		},
	}

	out := captureStdout(t, func() {
		printCallPatternReport(report)
	})

	if want := "warning [z-warning]"; !strings.Contains(out, want) {
		t.Fatalf("output missing %q:\n%s", want, out)
	}
	if want := "error   [a-error]"; !strings.Contains(out, want) {
		t.Fatalf("output missing %q:\n%s", want, out)
	}
}
