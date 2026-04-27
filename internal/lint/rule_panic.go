package lint

import (
	"go/ast"
	"strings"
)

func checkPanicUsage(ctx CheckContext) []Violation {
	n := ctx.FileName
	isService := strings.HasPrefix(n, "service") && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go")
	isRepo := strings.HasPrefix(n, "repository") && strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go")
	if !isService && !isRepo {
		return nil
	}
	var vs []Violation
	ast.Inspect(ctx.File, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Name != "panic" {
			return true
		}
		pos := ctx.Fset.Position(call.Pos())
		vs = append(vs, Violation{
			File:    ctx.FilePath,
			Line:    pos.Line,
			Message: "panic() is not allowed in service/repository code",
			Rule:    "no-panic",
		})
		return true
	})
	return vs
}
