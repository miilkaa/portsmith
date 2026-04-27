package lint

import (
	"fmt"
	"go/ast"
)

func checkContextFirstParam(ctx CheckContext) []Violation {
	if ctx.FileName != "service.go" && ctx.FileName != "repository.go" {
		return nil
	}
	var vs []Violation
	for _, decl := range ctx.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Type.Params == nil {
			continue
		}
		if len(fn.Recv.List) != 1 {
			continue
		}
		recvType := fn.Recv.List[0].Type
		recvName := receiverTypeName(recvType)
		if recvName != "Service" && recvName != "Repository" {
			continue
		}
		if !fn.Name.IsExported() {
			continue
		}
		params := fn.Type.Params.List
		if len(params) == 0 {
			continue
		}
		first := params[0].Type
		if isContextType(first) {
			continue
		}
		pos := ctx.Fset.Position(fn.Pos())
		vs = append(vs, Violation{
			File:    ctx.FilePath,
			Line:    pos.Line,
			Message: fmt.Sprintf("method %s should take context.Context as the first parameter", fn.Name.Name),
			Rule:    "context-first",
		})
	}
	return vs
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Context" {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "context" {
		return true
	}
	return false
}
