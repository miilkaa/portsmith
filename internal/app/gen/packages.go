package gen

import (
	"fmt"
	"time"

	"github.com/miilkaa/portsmith/internal/analyze"
	"github.com/miilkaa/portsmith/internal/project"
	"github.com/miilkaa/portsmith/internal/workpool"
	"golang.org/x/tools/go/packages"
)

// Caller scan context.

type callerScanContext struct {
	enabled    bool
	modulePath string
	modulePkgs []*packages.Package
}

func loadCallerScanContext(enabled bool) (callerScanContext, error) {
	if !enabled {
		return callerScanContext{}, nil
	}

	modulePath, err := analyze.DetectModulePath(".")
	if err != nil {
		return callerScanContext{}, fmt.Errorf("--scan-callers requires a go.mod in the current dir: %w", err)
	}

	modulePkgs, err := analyze.LoadModulePackages(".")
	if err != nil {
		return callerScanContext{}, fmt.Errorf("load module packages: %w", err)
	}

	return callerScanContext{
		enabled:    true,
		modulePath: modulePath,
		modulePkgs: modulePkgs,
	}, nil
}

// Package generation runner.

func generatePackages(
	dirs []string,
	configs map[string]project.Config,
	opts genOptions,
	callerCtx callerScanContext,
	progress *progressLogger,
) error {
	results := workpool.Run(dirs, func(_ int, dir string) (string, error) {
		progress.packageStart("generate", dir)
		started := time.Now()
		out, err := genPackage(
			dir,
			configs[dir],
			opts.dryRun,
			callerCtx.enabled,
			callerCtx.modulePath,
			callerCtx.modulePkgs,
		)
		progress.packageDone("generate", dir, started, err)
		return out, err
	})

	for _, result := range results {
		if result.Err != nil {
			return fmt.Errorf("%s: %w", result.Item, result.Err)
		}
		if opts.dryRun && result.Value != "" {
			fmt.Print(result.Value)
		}
	}
	return nil
}
