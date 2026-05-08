package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/miilkaa/portsmith/internal/lint"
)

func printReport(report checkReport) {
	sortViolations(report.warns)
	sortViolations(report.errs)

	for _, v := range report.warns {
		fmt.Printf("warning %-24s %s\n", "["+v.Rule+"]", v.String())
	}
	for _, v := range report.errs {
		fmt.Printf("error   %-24s %s\n", "["+v.Rule+"]", v.String())
	}
	if report.hasViolations() {
		fmt.Println()
		fmt.Print(violationSummary(report.errs, report.warns))
	}
}

// sortViolations sorts by rule name first, then file, then line.
func sortViolations(vs []lint.Violation) {
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].Rule != vs[j].Rule {
			return vs[i].Rule < vs[j].Rule
		}
		if vs[i].File != vs[j].File {
			return vs[i].File < vs[j].File
		}
		return vs[i].Line < vs[j].Line
	})
}

// violationSummary returns a one-line summary with per-rule counts.
func violationSummary(errs, warns []lint.Violation) string {
	counts := make(map[string]int)
	for _, v := range errs {
		counts[v.Rule]++
	}
	for _, v := range warns {
		counts[v.Rule]++
	}

	// Collect and sort rule names for deterministic output.
	rules := make([]string, 0, len(counts))
	for r := range counts {
		rules = append(rules, r)
	}
	sort.Strings(rules)

	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		parts = append(parts, fmt.Sprintf("%s: %d", r, counts[r]))
	}

	total := len(errs) + len(warns)
	return fmt.Sprintf("%d violation(s) — %s\n", total, strings.Join(parts, ", "))
}
