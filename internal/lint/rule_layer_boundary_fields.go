package lint

import (
	"fmt"
	"go/ast"
	"strings"
)

func checkLayerBoundaryFields(ctx CheckContext) []Violation {
	name := ctx.FileName
	isHandler := strings.HasPrefix(name, "handler") && name != "handler_test.go"
	isService := name == "service.go"
	if !isHandler && !isService {
		return nil
	}
	var vs []Violation
	for _, decl := range ctx.File.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			typeName := ts.Name.Name
			if typeName != "Handler" && typeName != "Service" {
				continue
			}
			for _, field := range st.Fields.List {
				// Rule 4: concrete *Service, *Repository, *Handler
				if star, ok := field.Type.(*ast.StarExpr); ok {
					if ident, ok := star.X.(*ast.Ident); ok {
						if ident.Name == "Service" || ident.Name == "Repository" || ident.Name == "Handler" {
							pos := ctx.Fset.Position(field.Pos())
							vs = append(vs, Violation{
								File:    ctx.FilePath,
								Line:    pos.Line,
								Message: fmt.Sprintf("concrete type *%s in struct %s — use an interface (port) instead", ident.Name, typeName),
								Rule:    "no-concrete-fields",
							})
						}
					}
				}
				base := typeBaseName(field.Type)
				if base == "" {
					continue
				}
				switch typeName {
				case "Handler":
					if ctx.Layers.Repository[base] {
						pos := ctx.Fset.Position(field.Pos())
						vs = append(vs, Violation{
							File:    ctx.FilePath,
							Line:    pos.Line,
							Message: fmt.Sprintf("handler field uses repository-layer type %s — depend on service port only", base),
							Rule:    "layer-dependency",
						})
					}
				case "Service":
					if ctx.Layers.Handler[base] {
						pos := ctx.Fset.Position(field.Pos())
						vs = append(vs, Violation{
							File:    ctx.FilePath,
							Line:    pos.Line,
							Message: fmt.Sprintf("service field uses handler-layer type %s — invalid dependency direction", base),
							Rule:    "layer-dependency",
						})
					}
				}
			}
		}
	}
	return vs
}

func typeBaseName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return typeBaseName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if t.Sel != nil {
			return t.Sel.Name
		}
		return ""
	default:
		return ""
	}
}
