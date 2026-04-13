// Package check implements the portsmith check command — an architecture linter
// that validates Clean Architecture rules across Go packages.
//
// Usage in CI/CD:
//
//	portsmith check ./internal/...
//
// Exit code 1 is returned when violations are found, 0 when clean.
// Output format is compatible with Go tools: file:line: message
package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/miilkaa/portsmith/internal/stack"
)

// Violation describes a single architectural rule violation.
type Violation struct {
	File    string
	Line    int
	Message string
}

func (v Violation) String() string {
	if v.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", v.File, v.Line, v.Message)
	}
	return fmt.Sprintf("%s: %s", v.File, v.Message)
}

// Run executes the check command for the given arguments.
// Arguments can be package directories or Go-style patterns (./internal/...).
func Run(args []string) error {
	rest, stackFlag := parseStackArgs(args)
	if len(rest) == 0 {
		rest = []string{"./..."}
	}

	stk, err := stack.ResolveFromWD(stackFlag)
	if err != nil {
		return err
	}
	fmt.Printf("  stack: %s\n", stk)

	dirs, err := resolveDirs(rest)
	if err != nil {
		return err
	}

	var allViolations []Violation
	for _, dir := range dirs {
		v, err := Violations(dir)
		if err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		allViolations = append(allViolations, v...)
	}

	for _, v := range allViolations {
		fmt.Println(v)
	}

	if len(allViolations) > 0 {
		return fmt.Errorf("%d architecture violation(s) found", len(allViolations))
	}
	return nil
}

func parseStackArgs(args []string) (rest []string, stackFlag string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--stack":
			if i+1 < len(args) {
				stackFlag = args[i+1]
				i++
			}
		default:
			rest = append(rest, a)
		}
	}
	return rest, stackFlag
}

// Violations checks a single package directory and returns all violations found.
// Returns an empty slice for clean packages.
func Violations(dir string) ([]Violation, error) {
	var violations []Violation

	// Rule 1: ports.go must exist when handler.go + service.go + repository.go are present.
	if hasFile(dir, "handler.go") && hasFile(dir, "service.go") && hasFile(dir, "repository.go") {
		if !hasFile(dir, "ports.go") {
			violations = append(violations, Violation{
				File:    filepath.Join(dir, "ports.go"),
				Message: "ports.go is missing — run: portsmith gen " + dir,
			})
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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

		v := checkFile(fset, f, path, e.Name())
		violations = append(violations, v...)
	}

	return violations, nil
}

func checkFile(fset *token.FileSet, f *ast.File, path, name string) []Violation {
	var violations []Violation

	isHandler := strings.HasPrefix(name, "handler") && name != "handler_test.go"
	isService := name == "service.go"

	// Rule 2: handler files must not import DB drivers directly (stack-specific surface).
	if isHandler {
		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if forbiddenHandlerDBImport(impPath) {
				pos := fset.Position(imp.Pos())
				violations = append(violations, Violation{
					File:    path,
					Line:    pos.Line,
					Message: fmt.Sprintf("handler imports %q directly — database access belongs in repository", impPath),
				})
			}
		}
	}

	// Rule 3: service files must not import HTTP or router frameworks.
	if isService {
		for _, imp := range f.Imports {
			impPath := strings.Trim(imp.Path.Value, `"`)
			if impPath == "net/http" ||
				strings.Contains(impPath, "gin-gonic/gin") ||
				strings.Contains(impPath, "go-chi/chi") {
				pos := fset.Position(imp.Pos())
				violations = append(violations, Violation{
					File:    path,
					Line:    pos.Line,
					Message: fmt.Sprintf("service imports %q — HTTP concerns belong in handler", impPath),
				})
			}
		}
	}

	// Rule 4: Handler and Service structs must use interfaces for service/repo fields, not concrete types.
	if isHandler || isService {
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
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				typeName := ts.Name.Name
				if typeName != "Handler" && typeName != "Service" {
					continue
				}
				for _, field := range st.Fields.List {
					if star, ok := field.Type.(*ast.StarExpr); ok {
						if ident, ok := star.X.(*ast.Ident); ok {
							// Concrete pointer to a struct — check if it looks like a layer type.
							if ident.Name == "Service" || ident.Name == "Repository" || ident.Name == "Handler" {
								pos := fset.Position(field.Pos())
								violations = append(violations, Violation{
									File:    path,
									Line:    pos.Line,
									Message: fmt.Sprintf("concrete type *%s in struct %s — use an interface (port) instead", ident.Name, typeName),
								})
							}
						}
					}
				}
			}
		}
	}

	return violations
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

// resolveDirs expands Go-style patterns like ./internal/... into directory paths.
func resolveDirs(patterns []string) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, pattern := range patterns {
		// Handle ./... style patterns.
		if strings.HasSuffix(pattern, "/...") {
			root := strings.TrimSuffix(pattern, "/...")
			root = strings.TrimPrefix(root, "./")
			err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return err
				}
				if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				if !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			continue
		}

		// Handle ./... at root.
		if pattern == "./..." {
			err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
				if err != nil || !d.IsDir() {
					return err
				}
				if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
					return filepath.SkipDir
				}
				if !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		if !seen[pattern] {
			seen[pattern] = true
			result = append(result, pattern)
		}
	}
	return result, nil
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
