package lint

import (
	"path/filepath"
)

func checkTestFilePresence(dir string) []Violation {
	var vs []Violation
	if hasFile(dir, "service.go") && !hasFile(dir, "service_test.go") {
		vs = append(vs, Violation{
			File:    filepath.Join(dir, "service.go"),
			Line:    0,
			Message: "service_test.go is missing — add tests alongside service.go",
			Rule:    "test-files",
		})
	}
	if hasFile(dir, "handler.go") && !hasFile(dir, "handler_test.go") {
		vs = append(vs, Violation{
			File:    filepath.Join(dir, "handler.go"),
			Line:    0,
			Message: "handler_test.go is missing — add tests alongside handler.go",
			Rule:    "test-files",
		})
	}
	return vs
}
