// Package check implements the portsmith check command — CLI entry for the architecture linter.
//
// Usage in CI/CD:
//
//	portsmith check ./internal/...
//
// Exit code 1 is returned when error-severity violations are found.
// Warning-severity violations are printed but do not fail the command.
package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/lintconfig"
	"github.com/miilkaa/portsmith/internal/stack"
	"github.com/miilkaa/portsmith/internal/workpool"
)

// Run executes the check command for the given arguments.
func Run(args []string) error {
	rest, stackFlag := parseStackArgs(args)
	if len(rest) == 0 {
		rest = []string{"./..."}
	}

	stk, err := stack.ResolveFromWD(stackFlag)
	if err != nil {
		return err
	}
	fmt.Printf("  stack: %s\n", stk)

	dirs, err := resolveDirs(rest)
	if err != nil {
		return err
	}

	results := workpool.Run(dirs, func(_ int, dir string) (checkResult, error) {
		root, err := stack.FindProjectRoot(dir)
		if err != nil {
			root = dir
		}
		cfg, lerr := lintconfig.Load(root)
		if lerr != nil {
			return checkResult{}, fmt.Errorf("%s: %w", root, lerr)
		}
		vs, verr := lint.Violations(dir, cfg, root)
		if verr != nil {
			return checkResult{}, verr
		}
		var result checkResult
		for _, v := range vs {
			switch cfg.Lint.RuleSeverity(v.Rule) {
			case lintconfig.SeverityOff:
				continue
			case lintconfig.SeverityWarning:
				result.warnViolations = append(result.warnViolations, v)
			default:
				result.errViolations = append(result.errViolations, v)
			}
		}
		return result, nil
	})

	var errViolations, warnViolations []lint.Violation
	for _, result := range results {
		if result.Err != nil {
			return fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		warnViolations = append(warnViolations, result.Value.warnViolations...)
		errViolations = append(errViolations, result.Value.errViolations...)
	}

	sortViolations(warnViolations)
	sortViolations(errViolations)

	for _, v := range warnViolations {
		fmt.Printf("warning %-24s %s\n", "["+v.Rule+"]", v.String())
	}
	for _, v := range errViolations {
		fmt.Printf("error   %-24s %s\n", "["+v.Rule+"]", v.String())
	}

	if len(errViolations) > 0 {
		fmt.Println()
		fmt.Print(violationSummary(errViolations, warnViolations))
		return fmt.Errorf("%d error violation(s)", len(errViolations))
	}
	if len(warnViolations) > 0 {
		fmt.Println()
		fmt.Print(violationSummary(errViolations, warnViolations))
	}
	return nil
}

type checkResult struct {
	errViolations  []lint.Violation
	warnViolations []lint.Violation
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

func parseStackArgs(args []string) (rest []string, stackFlag string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--stack":
			if i+1 < len(args) {
				stackFlag = args[i+1]
				i++
			}
		default:
			rest = append(rest, a)
		}
	}
	return rest, stackFlag
}

// resolveDirs expands Go-style patterns like ./internal/... into directory paths.
func resolveDirs(patterns []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, pattern := range patterns {
		if strings.HasSuffix(pattern, "/...") {
			root := strings.TrimSuffix(pattern, "/...")
			root = strings.TrimPrefix(root, "./")
			err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return err
				}
				if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				if !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}

		if pattern == "./..." {
			err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return err
				}
				if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				if !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		if !seen[pattern] {
			seen[pattern] = true
			result = append(result, pattern)
		}
	}
	return result, nil
}
