package lint

import (
	"os"
	"path/filepath"
)

func checkPortsPresence(dir string) []Violation {
	if !hasFile(dir, "handler.go") || !hasFile(dir, "service.go") || !hasFile(dir, "repository.go") {
		return nil
	}
	if hasFile(dir, "ports.go") {
		return nil
	}
	return []Violation{{
		File:    filepath.Join(dir, "ports.go"),
		Line:    0,
		Message: "ports.go is missing — run: portsmith gen " + dir,
			Rule:    "ports-required",
	}}
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
