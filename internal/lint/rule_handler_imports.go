package lint

import (
	"fmt"
	"strings"
)

func checkHandlerImports(ctx CheckContext) []Violation {
	name := ctx.FileName
	if !strings.HasPrefix(name, "handler") || name == "handler_test.go" {
		return nil
	}
	var vs []Violation
	for _, imp := range ctx.File.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		if forbiddenHandlerDBImport(impPath) {
			pos := ctx.Fset.Position(imp.Pos())
			vs = append(vs, Violation{
				File:    ctx.FilePath,
				Line:    pos.Line,
				Message: fmt.Sprintf("handler imports %q directly — database access belongs in repository", impPath),
				Rule:    "handler-no-db",
			})
		}
	}
	return vs
}

func forbiddenHandlerDBImport(impPath string) bool {
	if impPath == "database/sql" {
		return true
	}
	if strings.Contains(impPath, "gorm.io/gorm") {
		return true
	}
	if strings.Contains(impPath, "jmoiron/sqlx") {
		return true
	}
	return false
}
