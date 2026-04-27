package lint

import (
	"fmt"
	"strings"
)

func checkServiceImports(ctx CheckContext) []Violation {
	if ctx.FileName != "service.go" {
		return nil
	}
	var vs []Violation
	for _, imp := range ctx.File.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)
		if impPath == "net/http" ||
			strings.Contains(impPath, "gin-gonic/gin") ||
			strings.Contains(impPath, "go-chi/chi") {
			pos := ctx.Fset.Position(imp.Pos())
			vs = append(vs, Violation{
				File:    ctx.FilePath,
				Line:    pos.Line,
				Message: fmt.Sprintf("service imports %q — HTTP concerns belong in handler", impPath),
				Rule:    "service-no-http",
			})
		}
	}
	return vs
}
