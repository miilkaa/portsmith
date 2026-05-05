// Package lint implements portsmith architecture checks (rules R1–R14).
package lint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/miilkaa/portsmith/internal/gen"
	"github.com/miilkaa/portsmith/internal/lintconfig"
)

// Violations checks a single package directory and returns all violations (before severity filtering).
func Violations(dir string, cfg lintconfig.Config, projectRoot string) ([]Violation, error) {
	var vs []Violation
	vs = append(vs, checkPortsPresence(dir)...)
	vs = append(vs, checkPortsComplete(dir)...)
	vs = append(vs, checkTestFilePresence(dir)...)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	vs = append(vs, checkFileSizes(dir, projectRoot, cfg)...)

	modulePath, _ := gen.DetectModulePath(projectRoot)
	layers := discoverLayerTypes(dir)

	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		ctx := CheckContext{
			Dir:         dir,
			ProjectRoot: projectRoot,
			ModulePath:  modulePath,
			Fset:        fset,
			File:        f,
			FilePath:    path,
			FileName:    e.Name(),
			Layers:      layers,
			Config:      cfg,
		}
		vs = append(vs, checkFile(ctx)...)
	}

	vs = filterRulesOff(vs, cfg)
	return filterSuppressed(vs), nil
}

// CallPatternViolations checks only the call-pattern rule for a single package
// directory. It is used by portsmith gen before writing ports.go, where running
// the full linter would be too broad because some rules depend on generated
// ports.go already being current.
func CallPatternViolations(dir string, cfg lintconfig.Config) ([]Violation, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var vs []Violation
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		ctx := CheckContext{
			Dir:      dir,
			Fset:     fset,
			File:     f,
			FilePath: path,
			FileName: e.Name(),
			Config:   cfg,
		}
		vs = append(vs, checkCallPatterns(ctx)...)
	}

	vs = filterRulesOff(vs, cfg)
	return filterSuppressed(vs), nil
}

func filterRulesOff(vs []Violation, cfg lintconfig.Config) []Violation {
	var out []Violation
	for _, v := range vs {
		if cfg.Lint.RuleSeverity(v.Rule) == lintconfig.SeverityOff {
			continue
		}
		out = append(out, v)
	}
	return out
}

func checkFile(ctx CheckContext) []Violation {
	var vs []Violation
	vs = append(vs, checkHandlerImports(ctx)...)
	vs = append(vs, checkServiceImports(ctx)...)
	vs = append(vs, checkLayerBoundaryFields(ctx)...)
	vs = append(vs, checkExportedTypesInLayerFile(ctx)...)
	vs = append(vs, checkCrossModuleImports(ctx)...)
	vs = append(vs, checkConstructorInjection(ctx)...)
	vs = append(vs, checkPanicUsage(ctx)...)
	vs = append(vs, checkContextFirstParam(ctx)...)
	vs = append(vs, checkMethodCount(ctx)...)
	vs = append(vs, checkWiringViolations(ctx)...)
	vs = append(vs, checkLoggerRules(ctx)...)
	vs = append(vs, checkCallPatterns(ctx)...)
	return vs
}
