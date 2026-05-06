package lint

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/miilkaa/portsmith/internal/project"
)

func checkCallPatterns(ctx CheckContext) []Violation {
	name := ctx.FileName
	isHandler := strings.HasPrefix(name, "handler") && name != "handler_test.go"
	isService := name == "service.go"
	if !isHandler && !isService {
		return nil
	}

	var layerCfg project.LayerCallConfig
	var layerLabel string
	if isHandler {
		layerCfg = ctx.Config.Lint.CallPatterns.Handler
		layerLabel = "handler"
	} else {
		layerCfg = ctx.Config.Lint.CallPatterns.Service
		layerLabel = "service"
	}
	if !hasNonEmptyPattern(layerCfg.NotAllowed) {
		return nil
	}

	var vs []Violation
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		recv, field, method, ok := threeLevelSelectorCall(call.Fun)
		if !ok {
			return true
		}
		callKey := recv + "." + field + "." + method
		for _, raw := range layerCfg.NotAllowed {
			pat := strings.TrimSpace(raw)
			if pat == "" {
				continue
			}
			if matchCallPattern(recv, field, method, pat) {
				msg := fmt.Sprintf(
					`call pattern %q is not allowed in %s layer (not_allowed: %q)`,
					callKey, layerLabel, pat,
				)
				if hint := firstCallPatternHint(layerCfg.Allowed); hint != "" {
					msg += fmt.Sprintf(`; use %q instead`, hint)
				}
				pos := ctx.Fset.Position(call.Pos())
				vs = append(vs, Violation{
					File:    ctx.FilePath,
					Line:    pos.Line,
					Message: msg,
					Rule:    "call-pattern",
				})
				break
			}
		}
		return true
	})
	return vs
}

// threeLevelSelectorCall reports recv.field.method() from Fun = Sel(Sel(Ident, Ident), Ident).
func threeLevelSelectorCall(fun ast.Expr) (recv, field, method string, ok bool) {
	outer, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return "", "", "", false
	}
	inner, ok := outer.X.(*ast.SelectorExpr)
	if !ok {
		return "", "", "", false
	}
	recvIdent, ok := inner.X.(*ast.Ident)
	if !ok || inner.Sel == nil || outer.Sel == nil {
		return "", "", "", false
	}
	return recvIdent.Name, inner.Sel.Name, outer.Sel.Name, true
}

func matchCallSegment(pattern, ident string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == ident
}

func matchCallPattern(recv, field, method, pattern string) bool {
	pr, pf, pm, ok := splitCallPattern(pattern)
	if !ok {
		return false
	}
	return matchCallSegment(pr, recv) && matchCallSegment(pf, field) && matchCallSegment(pm, method)
}

func splitCallPattern(p string) (recv, field, method string, ok bool) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", "", "", false
	}
	parts := strings.Split(p, ".")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func firstCallPatternHint(allowed []string) string {
	for _, s := range allowed {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

func hasNonEmptyPattern(list []string) bool {
	for _, s := range list {
		if strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}
