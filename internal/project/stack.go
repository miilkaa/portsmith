package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Stack identifies HTTP + persistence choices for scaffolding and lint rules.
type Stack string

const (
	// GinGORM is the default: Gin HTTP + GORM.
	GinGORM Stack = "gin-gorm"
	// ChiSqlx is Chi HTTP + sqlx (e.g. PostgreSQL via pgx stdlib driver).
	ChiSqlx Stack = "chi-sqlx"
)

// FromFlag parses a --stack value.
func FromFlag(s string) (Stack, error) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "", string(GinGORM):
		return GinGORM, nil
	case "chi-sqlx":
		return ChiSqlx, nil
	default:
		return "", fmt.Errorf("unknown stack %q (use gin-gorm or chi-sqlx)", s)
	}
}

// FindProjectRoot walks upward from startDir until a directory containing go.mod is found.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		st, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", startDir)
		}
		dir = parent
	}
}

// Detect returns the stack for a project root (directory that contains go.mod).
// Priority: portsmith.yaml → go.mod heuristics → GinGORM.
func Detect(projectRoot string) Stack {
	cfg, err := Load(projectRoot)
	if err == nil && strings.TrimSpace(cfg.Stack) != "" {
		if s, err := FromFlag(cfg.Stack); err == nil {
			return s
		}
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		return GinGORM
	}
	mod := string(data)
	if strings.Contains(mod, "github.com/go-chi/chi") {
		return ChiSqlx
	}
	if strings.Contains(mod, "github.com/gin-gonic/gin") {
		return GinGORM
	}
	return GinGORM
}

// Resolve returns the stack to use: explicit flag wins, else Detect from project root of pkgDir.
func Resolve(pkgDir, stackFlag string) (Stack, error) {
	if strings.TrimSpace(stackFlag) != "" {
		return FromFlag(stackFlag)
	}
	root, err := FindProjectRoot(pkgDir)
	if err != nil {
		return GinGORM, nil
	}
	return Detect(root), nil
}

// ResolveFromWD is like Resolve but uses the current working directory as the start
// point for finding go.mod (for commands that are not tied to a single package path).
func ResolveFromWD(stackFlag string) (Stack, error) {
	if strings.TrimSpace(stackFlag) != "" {
		return FromFlag(stackFlag)
	}
	wd, err := os.Getwd()
	if err != nil {
		return GinGORM, nil
	}
	root, err := FindProjectRoot(wd)
	if err != nil {
		return GinGORM, nil
	}
	return Detect(root), nil
}
