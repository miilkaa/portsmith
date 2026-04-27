package lint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func checkCrossModuleImports(ctx CheckContext) []Violation {
	if ctx.ModulePath == "" || len(ctx.Config.Lint.AllowedCrossImports) == 0 {
		return nil
	}
	patterns := make([]string, 0, len(ctx.Config.Lint.AllowedCrossImports))
	for p := range ctx.Config.Lint.AllowedCrossImports {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	var allowed []string
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		ok, err := filepath.Match(pattern, ctx.FileName)
		if err != nil || !ok {
			continue
		}
		for _, a := range ctx.Config.Lint.AllowedCrossImports[pattern] {
			if !seen[a] {
				seen[a] = true
				allowed = append(allowed, a)
			}
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	prefix := ctx.ModulePath + "/internal/"
	var vs []Violation
	for _, imp := range ctx.File.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		ok := false
		for _, a := range allowed {
			a = strings.Trim(a, "/")
			full := ctx.ModulePath + "/" + a
			if strings.HasPrefix(path, full) || path == full {
				ok = true
				break
			}
		}
		if !ok {
			pos := ctx.Fset.Position(imp.Pos())
			vs = append(vs, Violation{
				File:    ctx.FilePath,
				Line:    pos.Line,
				Message: fmt.Sprintf("import %q not allowed by lint.allowed_cross_imports for this file", path),
				Rule:    "cross-imports",
			})
		}
	}
	return vs
}
