package lint

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/miilkaa/portsmith/internal/lintconfig"
)

func checkFileSizes(dir string, projectRoot string, cfg lintconfig.Config) []Violation {
	if len(cfg.Lint.MaxLines) == 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var vs []Violation
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		n := bytes.Count(data, []byte{'\n'})
		if len(data) > 0 && !bytes.HasSuffix(data, []byte{'\n'}) {
			n++
		}
		rel, _ := filepath.Rel(projectRoot, path)
		rel = filepath.ToSlash(rel)
		limit := resolveLineLimit(rel, e.Name(), cfg.Lint.MaxLines)
		if limit <= 0 || n <= limit {
			continue
		}
		vs = append(vs, Violation{
			File:    path,
			Line:    0,
			Message: fmt.Sprintf("file has %d lines (limit %d)", n, limit),
			Rule:    "file-size",
		})
	}
	return vs
}

func resolveLineLimit(relPath, base string, rules []lintconfig.FileSizeRule) int {
	var best int
	for _, r := range rules {
		if r.File != "" && filepath.ToSlash(r.File) == relPath {
			return r.Limit
		}
	}
	for _, r := range rules {
		if r.Pattern == "" {
			continue
		}
		ok, _ := filepath.Match(r.Pattern, base)
		if !ok {
			continue
		}
		if best == 0 || (r.Limit > 0 && r.Limit < best) {
			best = r.Limit
		}
	}
	return best
}
