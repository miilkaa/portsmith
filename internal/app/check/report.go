package check

import (
	"fmt"

	"github.com/miilkaa/portsmith/internal/app/violations"
)

func printReport(report checkReport) {
	violations.PrintLines(report.warns, report.errs)
	if report.hasViolations() {
		fmt.Println()
		fmt.Print(violations.Summary(report.errs, report.warns))
	}
}
