package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"
)

func checkExportedTypesInLayerFile(ctx CheckContext) []Violation {
	name := ctx.FileName
	var vs []Violation
	if strings.HasPrefix(name, "repository") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
		for _, decl := range ctx.File.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); !ok {
					continue
				}
				x := ts.Name.Name
				if x == "" || !unicode.IsUpper(rune(x[0])) {
					continue
				}
				if !strings.HasSuffix(x, "Repository") {
					pos := ctx.Fset.Position(ts.Pos())
					vs = append(vs, Violation{
						File:    ctx.FilePath,
						Line:    pos.Line,
						Message: fmt.Sprintf("exported type %s in repository file — move to dto.go or model.go", x),
						Rule:    "type-placement",
					})
				}
			}
		}
		return vs
	}
	if name == "service.go" {
		for _, decl := range ctx.File.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); !ok {
					continue
				}
				x := ts.Name.Name
				if x == "" || !unicode.IsUpper(rune(x[0])) {
					continue
				}
				if x != "Service" {
					pos := ctx.Fset.Position(ts.Pos())
					vs = append(vs, Violation{
						File:    ctx.FilePath,
						Line:    pos.Line,
						Message: fmt.Sprintf("exported type %s in service.go — move to dto.go or model.go", x),
						Rule:    "type-placement",
					})
				}
			}
		}
	}
	return vs
}
