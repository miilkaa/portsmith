package lint

import (
	"fmt"
	"go/ast"
	"strings"
)

func checkWiringViolations(ctx CheckContext) []Violation {
	allowed := ctx.Config.Lint.Wiring.AllowedFiles
	if len(allowed) == 0 {
		return nil
	}
	for _, f := range allowed {
		if f == ctx.FileName {
			return nil
		}
	}
	var vs []Violation
	ast.Inspect(ctx.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		name := id.Name
		if !strings.HasPrefix(name, "New") {
			return true
		}
		if strings.HasSuffix(name, "Repository") || strings.HasSuffix(name, "Service") || strings.HasSuffix(name, "Handler") {
			pos := ctx.Fset.Position(call.Pos())
			vs = append(vs, Violation{
				File:    ctx.FilePath,
				Line:    pos.Line,
				Message: fmt.Sprintf("call %s must live in wiring files %v", name, allowed),
				Rule:    "wiring-isolation",
			})
		}
		return true
	})
	return vs
}
