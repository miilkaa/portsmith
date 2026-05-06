package lint

import (
	"go/ast"
	"strings"
)

// knownLoggingImports lists third-party and std logging packages the linter recognizes.
// Only these are considered by logger-no-other; other imports are ignored.
var knownLoggingImports = map[string]struct{}{
	"log":                        {},
	"log/slog":                   {},
	"go.uber.org/zap":            {},
	"github.com/sirupsen/logrus": {},
	"github.com/rs/zerolog":      {},
}

func checkLoggerRules(ctx CheckContext) []Violation {
	allowed := strings.TrimSpace(ctx.Config.Lint.Logger.Allowed)
	if allowed == "" {
		return nil
	}
	var vs []Violation
	vs = append(vs, checkLoggerNoOther(ctx, allowed)...)
	vs = append(vs, checkLoggerNoFmtPrint(ctx)...)
	vs = append(vs, checkLoggerNoInit(ctx, allowed)...)
	return vs
}

func checkLoggerNoOther(ctx CheckContext, allowed string) []Violation {
	var vs []Violation
	for _, imp := range ctx.File.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if _, known := knownLoggingImports[path]; !known {
			continue
		}
		if path == allowed {
			continue
		}
		pos := ctx.Fset.Position(imp.Pos())
		vs = append(vs, Violation{
			File:    pos.Filename,
			Line:    pos.Line,
			Message: "import " + path + " is not the configured logger — use " + allowed + " instead",
			Rule:    "logger-no-other",
		})
	}
	return vs
}

func checkLoggerNoFmtPrint(ctx CheckContext) []Violation {
	var vs []Violation
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "fmt" {
			return true
		}
		name := sel.Sel.Name
		if !strings.HasPrefix(name, "Print") && !strings.HasPrefix(name, "Fprint") {
			return true
		}
		pos := ctx.Fset.Position(sel.Sel.Pos())
		vs = append(vs, Violation{
			File:    pos.Filename,
			Line:    pos.Line,
			Message: "fmt." + name + " — use the configured structured logger instead of fmt",
			Rule:    "logger-no-fmt-print",
		})
		return true
	})
	return vs
}

func checkLoggerNoInit(ctx CheckContext, allowed string) []Violation {
	idToPath := importPathsByIdent(ctx.File)
	var vs []Violation
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "New" {
			return true
		}
		pkgID, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		path, ok := idToPath[pkgID.Name]
		if !ok || path != allowed {
			return true
		}
		pos := ctx.Fset.Position(sel.Sel.Pos())
		vs = append(vs, Violation{
			File:    pos.Filename,
			Line:    pos.Line,
			Message: pkgID.Name + ".New — create the logger once at startup / wiring, not inside packages",
			Rule:    "logger-no-init",
		})
		return true
	})
	return vs
}

// importPathsByIdent maps local package identifier (import name) to import path.
func importPathsByIdent(f *ast.File) map[string]string {
	out := make(map[string]string)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var name string
		if imp.Name != nil {
			if imp.Name.Name == "." {
				continue
			}
			name = imp.Name.Name
		} else {
			name = importDefaultName(path)
		}
		out[name] = path
	}
	return out
}

func importDefaultName(impPath string) string {
	i := strings.LastIndex(impPath, "/")
	if i < 0 {
		return impPath
	}
	return impPath[i+1:]
}
