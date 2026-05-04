// Package gen implements the core logic for the portsmith gen command:
// AST-based method signature extraction and regex-based call collection.
// The generator reads existing handler/service/repository files and produces
// a minimal ports.go with only the interface methods that are actually used.
package gen

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// --- Call collection (regex-based) ---

// Receiver patterns for repository access.
var (
	repoCallRe    = regexp.MustCompile(`\b(?:s|sr|e|t|w)\.repo\.([A-Za-z0-9_]+)\(`)
	repoCallReAI  = regexp.MustCompile(`\bh\.repository\.([A-Za-z0-9_]+)\(`)
	svcCallRe     = regexp.MustCompile(`\b\w+\.service\.([A-Za-z0-9_]+)\(`)
	repoAliasRe   = regexp.MustCompile(`\b(\w+)\s*:?=\s*(?:s|sr|e|t|w)\.repo\b`)
	repoAliasReAI = regexp.MustCompile(`\b(\w+)\s*:?=\s*h\.repository\b`)
	svcAliasRe    = regexp.MustCompile(`\b(\w+)\s*:?=\s*\w+\.service\b`)

	// Template for matching exported method calls on a named variable.
	aliasExportedMethodCallT = `\b%s\.([A-Z][A-Za-z0-9_]*)\(`
)

// CollectRepoCalls returns the set of Repository method names called in src.
func CollectRepoCalls(src string) map[string]struct{} {
	return collectCalls(src,
		[]*regexp.Regexp{repoCallRe, repoCallReAI},
		collectRepoAliasNames(src))
}

// CollectServiceCalls returns the set of Service method names called in src.
func CollectServiceCalls(src string) map[string]struct{} {
	return collectCalls(src,
		[]*regexp.Regexp{svcCallRe},
		collectServiceAliasNames(src))
}

// collectCalls is the shared implementation behind CollectRepoCalls / CollectServiceCalls.
// It unions matches from each direct regex with exported-only matches via local aliases.
func collectCalls(src string, direct []*regexp.Regexp, aliases map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for _, re := range direct {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			out[m[1]] = struct{}{}
		}
	}
	for alias := range aliases {
		re := regexp.MustCompile(fmt.Sprintf(aliasExportedMethodCallT, regexp.QuoteMeta(alias)))
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			out[m[1]] = struct{}{}
		}
	}
	return out
}

func collectRepoAliasNames(src string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, re := range []*regexp.Regexp{repoAliasRe, repoAliasReAI} {
		for _, loc := range re.FindAllStringSubmatchIndex(src, -1) {
			if len(loc) < 4 {
				continue
			}
			if !isAliasRHS(src, loc[1]) {
				continue
			}
			out[src[loc[2]:loc[3]]] = struct{}{}
		}
	}
	return out
}

func collectServiceAliasNames(src string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, loc := range svcAliasRe.FindAllStringSubmatchIndex(src, -1) {
		if len(loc) < 4 {
			continue
		}
		if !isAliasRHS(src, loc[1]) {
			continue
		}
		out[src[loc[2]:loc[3]]] = struct{}{}
	}
	return out
}

// isAliasRHS checks that what follows the alias assignment is not a method call
// (i.e., it's a plain variable capture, not h.repo.Method()).
func isAliasRHS(src string, valueEnd int) bool {
	i := valueEnd
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if i >= len(src) {
		return true
	}
	switch src[i] {
	case '.', '(':
		return false
	case '\n', '\r', ';':
		return true
	case '/':
		return i+1 < len(src) && src[i+1] == '/'
	default:
		return true
	}
}

// --- AST-based method signature extraction ---

// Package is a minimal view of a parsed Go package: just the files we need.
// It replaces ast.Package, which was deprecated alongside parser.ParseDir.
type Package struct {
	Files []*ast.File
}

// MethodSigs parses pkg and returns a map of methodName → signature string
// for all exported methods on *typeName.
func MethodSigs(pkg *Package, typeName string) map[string]string {
	out := make(map[string]string)
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name == nil {
				continue
			}
			recv := fn.Recv.List
			if len(recv) != 1 {
				continue
			}
			star, ok := recv[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			id, ok := star.X.(*ast.Ident)
			if !ok || id.Name != typeName {
				continue
			}
			if !fn.Name.IsExported() {
				continue
			}
			out[fn.Name.Name] = sigString(fn)
		}
	}
	return out
}

// ParsePackage parses all non-test .go files in dir (excluding ports.go and adapters).
func ParsePackage(dir string) (*Package, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		if n == "ports.go" || n == "adapters.go" || strings.HasSuffix(n, "_adapter.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no package found in %s", dir)
	}
	return &Package{Files: files}, nil
}

// PackageName reads the package name from a Go source file.
func PackageName(path string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	return f.Name.Name, nil
}

// LoadSources reads non-test Go source files from dir as raw strings,
// excluding ports.go (the generator's output), adapters.go and *_adapter.go
// (cross-domain bridges that hold foreign-typed repo fields). The returned
// bodies feed into CollectRepoCalls / CollectServiceCalls which use regex
// matching on raw source text.
func LoadSources(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var bodies []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") || n == "ports.go" {
			continue
		}
		if n == "adapters.go" || strings.HasSuffix(n, "_adapter.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, string(b))
	}
	return bodies, nil
}

// SortSet returns the keys of a set map sorted alphabetically.
func SortSet(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// Union merges b into a and returns a.
func Union(a, b map[string]struct{}) map[string]struct{} {
	for k := range b {
		a[k] = struct{}{}
	}
	return a
}

// --- Port prefix ---

// PortPrefix returns the PascalCase prefix for interface names based on the directory name.
// "orders" → "Orders", "api_keys" → "ApiKeys".
func PortPrefix(dirBase string) string {
	parts := strings.Split(dirBase, "_")
	for i, s := range parts {
		if s == "" {
			continue
		}
		parts[i] = strings.ToUpper(s[:1]) + s[1:]
	}
	return strings.Join(parts, "")
}

// InferPortPrefix derives the interface name prefix from Handler and Service struct fields
// when dependency types are named like WebPushRepository / WebPushService.
//
// This fixes directory names such as "webpush" where PortPrefix yields "Webpush" while
// the codebase uses the idiomatic "WebPush".
func InferPortPrefix(pkg *Package) (string, bool) {
	var repoPfx, svcPfx string
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
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
				switch ts.Name.Name {
				case "Handler", "Service":
				default:
					continue
				}
				for _, field := range st.Fields.List {
					name, ok := typeNameForPortInference(field.Type)
					if !ok {
						continue
					}
					if p, ok := prefixFromPortTypeName(name, "Repository"); ok {
						repoPfx = p
					}
					if p, ok := prefixFromPortTypeName(name, "Service"); ok {
						svcPfx = p
					}
				}
			}
		}
	}
	switch {
	case repoPfx != "" && svcPfx != "":
		if repoPfx == svcPfx {
			return repoPfx, true
		}
		return "", false
	case repoPfx != "":
		return repoPfx, true
	case svcPfx != "":
		return svcPfx, true
	default:
		return "", false
	}
}

func prefixFromPortTypeName(typeName, suffix string) (string, bool) {
	if !strings.HasSuffix(typeName, suffix) {
		return "", false
	}
	base := strings.TrimSuffix(typeName, suffix)
	if base == "" {
		return "", false
	}
	return base, true
}

func typeNameForPortInference(expr ast.Expr) (string, bool) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return typeNameForPortInference(t.X)
	case *ast.Ident:
		return t.Name, true
	case *ast.SelectorExpr:
		return t.Sel.Name, true
	default:
		return "", false
	}
}

// --- Module path detection ---

// DetectModulePath reads the module directive from go.mod in root.
func DetectModulePath(root string) (string, error) {
	gomod := filepath.Join(root, "go.mod")
	f, err := os.Open(gomod)
	if err != nil {
		return "", fmt.Errorf("gen: go.mod not found in %s: %w", root, err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", fmt.Errorf("gen: module directive not found in %s", gomod)
}

// --- Signature formatting ---

func sigString(fn *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString(fn.Name.Name)
	b.WriteString(formatParams(fn.Type.Params))
	if fn.Type.Results != nil {
		b.WriteString(" ")
		b.WriteString(formatResults(fn.Type.Results))
	}
	return b.String()
}

func formatParams(fl *ast.FieldList) string {
	if fl == nil {
		return "()"
	}
	var parts []string
	for _, f := range fl.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		for _, n := range f.Names {
			parts = append(parts, n.Name+" "+typ)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func formatResults(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	if len(fl.List) == 1 && len(fl.List[0].Names) == 0 {
		return exprString(fl.List[0].Type)
	}
	var parts []string
	for _, f := range fl.List {
		typ := exprString(f.Type)
		if len(f.Names) == 0 {
			parts = append(parts, typ)
			continue
		}
		for _, n := range f.Names {
			parts = append(parts, n.Name+" "+typ)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		if len(t.Methods.List) == 0 {
			return "interface{}"
		}
	case *ast.FuncType:
		params := "()"
		if t.Params != nil {
			params = formatParams(t.Params)
		}
		results := ""
		if t.Results != nil {
			results = " " + formatResults(t.Results)
		}
		return "func" + params + results
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + exprString(t.Value)
		case ast.RECV:
			return "<-chan " + exprString(t.Value)
		default:
			return "chan " + exprString(t.Value)
		}
	case *ast.IndexExpr:
		return exprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.IndexListExpr:
		var idx []string
		for _, ix := range t.Indices {
			idx = append(idx, exprString(ix))
		}
		return exprString(t.X) + "[" + strings.Join(idx, ", ") + "]"
	case *ast.StructType:
		return "struct{}"
	case *ast.BasicLit:
		return t.Value
	}
	return "any /* portsmith: fix signature manually */"
}
