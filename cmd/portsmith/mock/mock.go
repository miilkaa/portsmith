// Package mockcmd implements the portsmith mock command.
// It wraps mockery to generate mocks for all interfaces in ports.go.
package mockcmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
)

// Run executes the mock command for the given package directories.
func Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: portsmith mock <pkg-dir> [<pkg-dir>...]")
	}

	// Verify mockery is available.
	if _, err := exec.LookPath("mockery"); err != nil {
		return fmt.Errorf("mockery not found in PATH. Install with: go install github.com/vektra/mockery/v2@latest")
	}

	for _, dir := range args {
		if err := mockPackage(dir); err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
	}
	return nil
}

func mockPackage(dir string) error {
	portsFile := filepath.Join(dir, "ports.go")
	if _, err := os.Stat(portsFile); err != nil {
		return fmt.Errorf("ports.go not found — run portsmith gen %s first", dir)
	}

	interfaces, err := parseInterfaces(portsFile)
	if err != nil {
		return fmt.Errorf("parse ports.go: %w", err)
	}

	outDir := filepath.Join(dir, "mocks")
	for _, iface := range interfaces {
		fmt.Printf("  mock %s → %s/mock_%s.go\n", iface, outDir, toLower(iface))
		cmd := exec.Command("mockery",
			"--name="+iface,
			"--dir="+dir,
			"--output="+outDir,
			"--outpkg=mocks",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("mockery %s: %w", iface, err)
		}
	}
	return nil
}

// parseInterfaces returns all exported interface names from a Go file.
func parseInterfaces(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var names []string
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
			if _, ok := ts.Type.(*ast.InterfaceType); !ok {
				continue
			}
			if ts.Name.IsExported() {
				names = append(names, ts.Name.Name)
			}
		}
	}
	return names, nil
}

func toLower(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]+32) + s[1:]
}
