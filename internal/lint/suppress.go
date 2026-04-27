package lint

import (
	"os"
	"strings"
)

// filterSuppressed removes violations covered by //nolint:portsmith in source files.
func filterSuppressed(vs []Violation) []Violation {
	cache := make(map[string]map[int]suppressEntry)
	var out []Violation
	for _, v := range vs {
		m, ok := cache[v.File]
		if !ok {
			m = parseNolintFile(v.File)
			cache[v.File] = m
		}
		if suppressedAt(m, v.Line, v.Rule) {
			continue
		}
		out = append(out, v)
	}
	return out
}

type suppressEntry struct {
	all   bool
	rules map[string]bool
}

func parseNolintFile(path string) map[int]suppressEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	out := make(map[int]suppressEntry)
	for i, line := range lines {
		lineNum := i + 1
		idx := strings.Index(line, "//nolint:portsmith")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+len("//nolint:portsmith"):])
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		var ent suppressEntry
		if rest == "" || rest == "all" {
			ent.all = true
		} else {
			ent.rules = make(map[string]bool)
			for _, p := range strings.Split(rest, ",") {
				p = strings.TrimSpace(strings.ToLower(p))
				if p != "" {
					ent.rules[p] = true
				}
			}
		}
		codeBefore := strings.TrimSpace(line[:idx])
		target := lineNum + 1
		if codeBefore != "" {
			target = lineNum
		}
		out[target] = ent
	}
	return out
}

func suppressedAt(m map[int]suppressEntry, line int, rule string) bool {
	if m == nil {
		return false
	}
	ent, ok := m[line]
	if !ok {
		return false
	}
	if ent.all {
		return true
	}
	rule = strings.ToLower(strings.TrimSpace(rule))
	return ent.rules[rule]
}
