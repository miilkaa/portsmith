package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// checkPortsComplete reports exported methods on *Service and *Repository types
// that are declared in implementation files but absent from the corresponding
// interface in ports.go.
//
// This catches methods that exist on the struct but were never called through
// the port — so portsmith gen did not include them — which can indicate either
// dead code or a missing wiring step.
func checkPortsComplete(dir string) []Violation {
	if !hasFile(dir, "ports.go") {
		return nil // ports-required covers the missing-file case
	}

	fset := token.NewFileSet()
	portsAST, err := parser.ParseFile(fset, filepath.Join(dir, "ports.go"), nil, 0)
	if err != nil {
		return nil
	}

	// interface name suffix → set of method names declared in that interface.
	ifaceMethods := collectPortsInterfaceMethods(portsAST)
	if len(ifaceMethods) == 0 {
		return nil
	}

	// impl type suffix → method name → source position.
	implMethods := collectImplMethods(dir)

	var vs []Violation
	for suffix, methods := range implMethods {
		iface, ok := ifaceMethods[suffix]
		if !ok {
			continue
		}
		for name, pos := range methods {
			if _, present := iface[name]; !present {
				vs = append(vs, Violation{
					File: pos.filename,
					Line: pos.line,
					Message: name + " (*" + suffix + ") is not in the ports.go interface" +
						" — add it to the interface or remove the method",
					Rule: "ports-complete",
				})
			}
		}
	}
	return vs
}

// collectPortsInterfaceMethods parses ports.go and returns a map from
// layer-suffix ("Service", "Repository") to the set of method names in
// all interfaces whose name ends with that suffix.
func collectPortsInterfaceMethods(f *ast.File) map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{})
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			suffix := layerTypeSuffix(ts.Name.Name)
			if suffix == "" {
				continue
			}
			if out[suffix] == nil {
				out[suffix] = make(map[string]struct{})
			}
			for _, m := range iface.Methods.List {
				for _, name := range m.Names {
					out[suffix][name.Name] = struct{}{}
				}
			}
		}
	}
	return out
}

type implMethodPos struct {
	filename string
	line     int
}

// collectImplMethods scans non-test .go files (excluding ports.go) and
// returns exported methods grouped by receiver type suffix.
func collectImplMethods(dir string) map[string]map[string]implMethodPos {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make(map[string]map[string]implMethodPos)
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() ||
			!strings.HasSuffix(e.Name(), ".go") ||
			strings.HasSuffix(e.Name(), "_test.go") ||
			e.Name() == "ports.go" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 || !fd.Name.IsExported() {
				continue
			}
			typeName := receiverTypeName(fd.Recv.List[0].Type)
			suffix := layerTypeSuffix(typeName)
			if suffix == "" {
				continue
			}
			if out[suffix] == nil {
				out[suffix] = make(map[string]implMethodPos)
			}
			pos := fset.Position(fd.Name.Pos())
			out[suffix][fd.Name.Name] = implMethodPos{filename: pos.Filename, line: pos.Line}
		}
	}
	return out
}

// layerTypeSuffix returns "Service" or "Repository" if the name ends with
// one of those suffixes, otherwise "".
func layerTypeSuffix(name string) string {
	for _, s := range []string{"Service", "Repository"} {
		if strings.HasSuffix(name, s) {
			return s
		}
	}
	return ""
}
