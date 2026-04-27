package lint

import (
	"fmt"
	"go/ast"
	"strings"
)

func checkConstructorInjection(ctx CheckContext) []Violation {
	name := ctx.FileName
	isHandler := strings.HasPrefix(name, "handler") && name != "handler_test.go"
	isService := name == "service.go"
	if !isHandler && !isService {
		return nil
	}
	var vs []Violation
	for _, decl := range ctx.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !strings.HasPrefix(fn.Name.Name, "New") {
			continue
		}
		if fn.Type.Params == nil {
			continue
		}
		for _, field := range fn.Type.Params.List {
			expr := field.Type
			base := typeBaseName(expr)
			pos := ctx.Fset.Position(field.Pos())

			if isHandler {
				if ctx.Layers.Repository[base] {
					vs = append(vs, Violation{
						File:    ctx.FilePath,
						Line:    pos.Line,
						Message: fmt.Sprintf("constructor %s must not accept repository-layer type %s — use a service port", fn.Name.Name, base),
						Rule:    "constructor-injection",
					})
				}
				if star, ok := expr.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok && ctx.Layers.Service[id.Name] {
						vs = append(vs, Violation{
							File:    ctx.FilePath,
							Line:    pos.Line,
							Message: fmt.Sprintf("constructor %s must not accept concrete *%s — use a service interface port", fn.Name.Name, id.Name),
							Rule:    "constructor-injection",
						})
					}
				}
			}
			if isService {
				if star, ok := expr.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok && ctx.Layers.Repository[id.Name] {
						vs = append(vs, Violation{
							File:    ctx.FilePath,
							Line:    pos.Line,
							Message: fmt.Sprintf("constructor %s must not accept concrete *%s — use a repository interface port", fn.Name.Name, id.Name),
							Rule:    "constructor-injection",
						})
					}
				}
			}
		}
	}
	return vs
}
