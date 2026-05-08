package check

import "github.com/miilkaa/portsmith/internal/lint"

type packageResult struct {
	errViolations  []lint.Violation
	warnViolations []lint.Violation
}

type checkReport struct {
	errs  []lint.Violation
	warns []lint.Violation
}

func (r *checkReport) addPackage(result packageResult) {
	r.warns = append(r.warns, result.warnViolations...)
	r.errs = append(r.errs, result.errViolations...)
}

func (r *checkReport) hasErrors() bool {
	return len(r.errs) > 0
}

func (r *checkReport) errorCount() int {
	return len(r.errs)
}

func (r *checkReport) hasViolations() bool {
	return len(r.errs) > 0 || len(r.warns) > 0
}
