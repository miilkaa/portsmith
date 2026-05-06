// Package scaffold implements the portsmith new command.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/miilkaa/portsmith/internal/project"
)

// Run executes the new command to scaffold a new package.
func Run(args []string) error {
	positional, stackFlag, err := parseArgs(args)
	if err != nil {
		return err
	}
	if len(positional) == 0 {
		return fmt.Errorf("usage: portsmith new [--stack gin-gorm|chi-sqlx] <pkg-dir>")
	}
	dir := positional[0]
	base := filepath.Base(dir)
	name := toPascalCase(base)

	pkgDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	st, err := project.Resolve(pkgDir, stackFlag)
	if err != nil {
		return err
	}
	fmt.Printf("  stack: %s\n", st)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	pkgName := toPackageName(base)
	data := templateData{
		Package: pkgName,
		Name:    name,
		Dir:     dir,
	}

	tpls := templatesFor(st)
	files := []struct {
		name    string
		content string
	}{
		{"domain.go", tpls.domain},
		{"errors.go", errorsTpl},
		{"ports.go", tpls.ports},
		{"service.go", tpls.service},
		{"repository.go", tpls.repo},
		{"handler.go", tpls.handler},
		{"dto.go", tpls.dto},
	}

	for _, f := range files {
		path := filepath.Join(dir, f.name)
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  skip  %s (already exists)\n", path)
			continue
		}
		if err := writeTemplate(path, f.content, data); err != nil {
			return fmt.Errorf("write %s: %w", f.name, err)
		}
		fmt.Printf("  create %s\n", path)
	}

	fmt.Printf("\nPackage scaffolded. Next steps:\n")
	fmt.Printf("  1. Add your domain fields to %s/domain.go\n", dir)
	fmt.Printf("  2. Implement service methods in %s/service.go\n", dir)
	fmt.Printf("  3. Run: portsmith gen %s\n", dir)
	fmt.Printf("  4. Run: portsmith mock %s\n", dir)
	return nil
}

func parseArgs(args []string) (positional []string, stackFlag string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--stack":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("--stack requires a value (gin-gorm or chi-sqlx)")
			}
			stackFlag = args[i+1]
			i++
		default:
			positional = append(positional, a)
		}
	}
	return positional, stackFlag, nil
}

type templateData struct {
	Package string
	Name    string
	Dir     string
}

type stackTemplates struct {
	domain, ports, service, repo, handler, dto string
}

func templatesFor(st project.Stack) stackTemplates {
	switch st {
	case project.ChiSqlx:
		return stackTemplates{
			domain:  domainTplChi,
			ports:   portsTplChi,
			service: serviceTplChi,
			repo:    repoTplChi,
			handler: handlerTplChi,
			dto:     dtoTplChi,
		}
	default:
		return stackTemplates{
			domain:  domainTplGin,
			ports:   portsTplGin,
			service: serviceTplGin,
			repo:    repoTplGin,
			handler: handlerTplGin,
			dto:     dtoTplGin,
		}
	}
}

func writeTemplate(path, tplContent string, data templateData) error {
	tpl, err := template.New("").Parse(tplContent)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

func toPascalCase(s string) string {
	parts := identifierParts(s)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	out := strings.Join(parts, "")
	if out == "" {
		return "Package"
	}
	if startsWithDigit(out) {
		return "P" + out
	}
	return out
}

func toPackageName(s string) string {
	parts := identifierParts(s)
	if len(parts) == 0 {
		return "package"
	}
	out := strings.ToLower(strings.Join(parts, ""))
	if startsWithDigit(out) {
		return "p" + out
	}
	return out
}

func identifierParts(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func startsWithDigit(s string) bool {
	for _, r := range s {
		return unicode.IsDigit(r)
	}
	return false
}
