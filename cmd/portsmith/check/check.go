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
	"strings"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/lintconfig"
	"github.com/miilkaa/portsmith/internal/stack"
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

	var errViolations, warnViolations []lint.Violation
	for _, dir := range dirs {
		root, err := stack.FindProjectRoot(dir)
		if err != nil {
			root = dir
		}
		cfg, lerr := lintconfig.Load(root)
		if lerr != nil {
			return fmt.Errorf("%s: %w", root, lerr)
		}
		vs, verr := lint.Violations(dir, cfg, root)
		if verr != nil {
			return fmt.Errorf("%s: %w", dir, verr)
		}
		for _, v := range vs {
			switch cfg.Lint.RuleSeverity(v.Rule) {
			case lintconfig.SeverityOff:
				continue
			case lintconfig.SeverityWarning:
				warnViolations = append(warnViolations, v)
			default:
				errViolations = append(errViolations, v)
			}
		}
	}

	for _, v := range warnViolations {
		fmt.Printf("warning: %s\n", v.String())
	}
	for _, v := range errViolations {
		fmt.Println(v.String())
	}

	if len(errViolations) > 0 {
		return fmt.Errorf("%d architecture violation(s) found", len(errViolations))
	}
	return nil
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
