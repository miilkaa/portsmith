package gen

import (
	"fmt"
	"time"

	"github.com/miilkaa/portsmith/internal/lint"
	"github.com/miilkaa/portsmith/internal/project"
	"github.com/miilkaa/portsmith/internal/workpool"
)

type callPatternResult struct {
	cfg            project.Config
	errViolations  []lint.Violation
	warnViolations []lint.Violation
}

func checkCallPatternsBeforeGen(dirs []string, progress *progressLogger) (map[string]project.Config, error) {
	configs := make(map[string]project.Config, len(dirs))
	var errViolations, warnViolations []lint.Violation
	results := workpool.Run(dirs, func(_ int, dir string) (callPatternResult, error) {
		progress.packageStart("call-pattern", dir)
		started := time.Now()
		root, err := project.FindProjectRoot(dir)
		if err != nil {
			root = dir
		}
		cfg, err := project.Load(root)
		if err != nil {
			err = fmt.Errorf("%s: %w", root, err)
			progress.packageDone("call-pattern", dir, started, err)
			return callPatternResult{}, err
		}
		vs, err := lint.CallPatternViolations(dir, cfg)
		if err != nil {
			progress.packageDone("call-pattern", dir, started, err)
			return callPatternResult{}, err
		}
		result := callPatternResult{cfg: cfg}
		for _, v := range vs {
			switch cfg.Lint.RuleSeverity(v.Rule) {
			case project.SeverityWarning:
				result.warnViolations = append(result.warnViolations, v)
			case project.SeverityOff:
				continue
			default:
				result.errViolations = append(result.errViolations, v)
			}
		}
		progress.packageDone("call-pattern", dir, started, nil)
		return result, nil
	})

	for _, result := range results {
		if result.Err != nil {
			return nil, fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		configs[result.Item] = result.Value.cfg
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
		return nil, fmt.Errorf("call-pattern check failed: %d error violation(s)", len(errViolations))
	}
	return configs, nil
}
