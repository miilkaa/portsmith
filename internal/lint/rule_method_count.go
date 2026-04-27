package lint

import (
	"fmt"
	"go/ast"
	"path/filepath"
	"strings"
)

func checkMethodCount(ctx CheckContext) []Violation {
	if len(ctx.Config.Lint.MaxMethods) == 0 {
		return nil
	}
	svc := countExportedMethods(ctx.File, "Service")
	hdl := countExportedMethods(ctx.File, "Handler")
	var vs []Violation
	for _, r := range ctx.Config.Lint.MaxMethods {
		if r.Limit <= 0 || r.Pattern == "" {
			continue
		}
		ok, err := filepath.Match(r.Pattern, ctx.FileName)
		if err != nil || !ok {
			continue
		}
		wantSvc := strings.Contains(r.Pattern, "service")
		wantHdl := strings.Contains(r.Pattern, "handler")
		switch {
		case wantSvc && !wantHdl:
			if svc > r.Limit {
				vs = append(vs, Violation{
					File: ctx.FilePath, Line: 0,
					Message: fmt.Sprintf("Service has %d exported methods (limit %d) — consider splitting responsibilities", svc, r.Limit),
					Rule:    "method-count",
				})
			}
		case wantHdl && !wantSvc:
			if hdl > r.Limit {
				vs = append(vs, Violation{
					File: ctx.FilePath, Line: 0,
					Message: fmt.Sprintf("Handler has %d exported methods (limit %d) — consider splitting responsibilities", hdl, r.Limit),
					Rule:    "method-count",
				})
			}
		default:
			if svc > r.Limit {
				vs = append(vs, Violation{
					File: ctx.FilePath, Line: 0,
					Message: fmt.Sprintf("Service has %d exported methods (limit %d) — consider splitting responsibilities", svc, r.Limit),
					Rule:    "method-count",
				})
			}
			if hdl > r.Limit {
				vs = append(vs, Violation{
					File: ctx.FilePath, Line: 0,
					Message: fmt.Sprintf("Handler has %d exported methods (limit %d) — consider splitting responsibilities", hdl, r.Limit),
					Rule:    "method-count",
				})
			}
		}
	}
	return vs
}

func countExportedMethods(f *ast.File, typeName string) int {
	n := 0
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}
		if len(fn.Recv.List) != 1 {
			continue
		}
		if receiverTypeName(fn.Recv.List[0].Type) != typeName {
			continue
		}
		n++
	}
	return n
}
