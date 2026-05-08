package check

import (
	"fmt"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/project"
)

func checkPackage(dir string) (packageResult, error) {
	root, err := project.FindProjectRoot(dir)
	if err != nil {
		root = dir
	}

	cfg, lerr := project.Load(root)
	if lerr != nil {
		return packageResult{}, fmt.Errorf("%s: %w", root, lerr)
	}

	vs, verr := lint.Violations(dir, cfg, root)
	if verr != nil {
		return packageResult{}, verr
	}

	var result packageResult
	for _, v := range vs {
		switch cfg.Lint.RuleSeverity(v.Rule) {
		case project.SeverityOff:
			continue
		case project.SeverityWarning:
			result.warnViolations = append(result.warnViolations, v)
		default:
			result.errViolations = append(result.errViolations, v)
		}
	}
	return result, nil
}
