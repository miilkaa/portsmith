package gen

import (
	"fmt"
	"time"

	"github.com/miilkaa/portsmith/internal/app/violations"
	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/project"
	"github.com/miilkaa/portsmith/internal/workpool"
)

// Call-pattern precheck.

func checkCallPatternsBeforeGen(dirs []string, progress *progressLogger) (map[string]project.Config, error) {
	report := newCallPatternReport(len(dirs))

	results := workpool.Run(dirs, func(_ int, dir string) (callPatternResult, error) {
		return checkCallPatternsPackage(dir, progress)
	})

	for _, result := range results {
		if result.Err != nil {
			return nil, fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		report.addPackage(result.Item, result.Value)
	}

	printCallPatternReport(report)
	if report.hasErrors() {
		return nil, fmt.Errorf("call-pattern check failed: %d error violation(s)", report.errorCount())
	}
	return report.configs, nil
}

// Package inspection.

type callPatternResult struct {
	cfg            project.Config
	errViolations  []lint.Violation
	warnViolations []lint.Violation
}

func checkCallPatternsPackage(dir string, progress *progressLogger) (callPatternResult, error) {
	progress.packageStart("call-pattern", dir)
	started := time.Now()

	result, err := inspectCallPatternsPackage(dir)
	progress.packageDone("call-pattern", dir, started, err)
	return result, err
}

func inspectCallPatternsPackage(dir string) (callPatternResult, error) {
	root, err := project.FindProjectRoot(dir)
	if err != nil {
		root = dir
	}

	cfg, err := project.Load(root)
	if err != nil {
		return callPatternResult{}, fmt.Errorf("%s: %w", root, err)
	}

	vs, err := lint.CallPatternViolations(dir, cfg)
	if err != nil {
		return callPatternResult{}, err
	}

	return classifyCallPatternViolations(cfg, vs), nil
}

func classifyCallPatternViolations(cfg project.Config, vs []lint.Violation) callPatternResult {
	result := callPatternResult{cfg: cfg}
	for _, v := range vs {
		result.addViolation(cfg, v)
	}
	return result
}

func (r *callPatternResult) addViolation(cfg project.Config, v lint.Violation) {
	switch cfg.Lint.RuleSeverity(v.Rule) {
	case project.SeverityWarning:
		r.warnViolations = append(r.warnViolations, v)
	case project.SeverityOff:
		return
	default:
		r.errViolations = append(r.errViolations, v)
	}
}

// Report.

type callPatternReport struct {
	configs map[string]project.Config
	errs    []lint.Violation
	warns   []lint.Violation
}

func newCallPatternReport(size int) callPatternReport {
	return callPatternReport{configs: make(map[string]project.Config, size)}
}

func (r *callPatternReport) addPackage(dir string, result callPatternResult) {
	r.configs[dir] = result.cfg
	r.warns = append(r.warns, result.warnViolations...)
	r.errs = append(r.errs, result.errViolations...)
}

func (r callPatternReport) hasErrors() bool {
	return len(r.errs) > 0
}

func (r callPatternReport) errorCount() int {
	return len(r.errs)
}

func printCallPatternReport(report callPatternReport) {
	violations.PrintLines(report.warns, report.errs)
}
