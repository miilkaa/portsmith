package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// LayerTypes maps type names to the architectural layer they belong to in this package.
type LayerTypes struct {
	Repository map[string]bool
	Service    map[string]bool
	Handler    map[string]bool
}

func newLayerTypes() LayerTypes {
	return LayerTypes{
		Repository: make(map[string]bool),
		Service:    make(map[string]bool),
		Handler:    make(map[string]bool),
	}
}

// discoverLayerTypes scans all non-test .go files in dir and builds layer membership.
func discoverLayerTypes(dir string) LayerTypes {
	lt := newLayerTypes()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return lt
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		name := e.Name()
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
				typeName := ts.Name.Name
				_, isStruct := ts.Type.(*ast.StructType)
				_, isIface := ts.Type.(*ast.InterfaceType)
				if !isStruct && !isIface {
					continue
				}
				switch {
				case name == "repository.go" && isStruct:
					lt.Repository[typeName] = true
				case name == "service.go" && isStruct:
					lt.Service[typeName] = true
				case strings.HasPrefix(name, "handler") && name != "handler_test.go" && isStruct:
					lt.Handler[typeName] = true
				}
			}
		}
	}
	// Suffix fallback: interfaces and types from ports.go etc.
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
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
				n := ts.Name.Name
				if strings.HasSuffix(n, "Repository") {
					lt.Repository[n] = true
				}
				if strings.HasSuffix(n, "Service") {
					lt.Service[n] = true
				}
				if strings.HasSuffix(n, "Handler") {
					lt.Handler[n] = true
				}
			}
		}
	}
	// Concrete Service/Repository/Handler struct names are already added in first pass.
	return lt
}
