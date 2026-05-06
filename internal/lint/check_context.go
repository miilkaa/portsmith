package lint

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/miilkaa/portsmith/internal/project"
)

// Violation describes a single architectural rule violation.
type Violation struct {
	File    string
	Line    int
	Message string
	Rule    string
}

func (v Violation) String() string {
	if v.Line > 0 {
		return v.File + ":" + strconv.Itoa(v.Line) + ": " + v.Message
	}
	return v.File + ": " + v.Message
}

// CheckContext is passed to each rule for one parsed file.
type CheckContext struct {
	Dir         string
	ProjectRoot string
	ModulePath  string
	Fset        *token.FileSet
	File        *ast.File
	FilePath    string
	FileName    string
	Layers      LayerTypes
	Config      project.Config
}
