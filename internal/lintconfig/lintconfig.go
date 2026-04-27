// Package lintconfig loads portsmith.yaml including lint rules and severity.
package lintconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root portsmith.yaml document.
type Config struct {
	Stack string     `yaml:"stack"`
	Lint  LintConfig `yaml:"lint"`
}

// LintConfig holds optional lint settings.
type LintConfig struct {
	MaxLines            []FileSizeRule        `yaml:"max_lines"`
	MaxMethods          []MaxMethodsRule      `yaml:"max_methods"`
	AllowedCrossImports map[string][]string   `yaml:"allowed_cross_imports"`
	Wiring              WiringConfig          `yaml:"wiring"`
	Logger              LoggerConfig          `yaml:"logger"`
	Rules               map[string]RuleConfig `yaml:"rules"`
}

// LoggerConfig enables logging-related lint rules when Allowed is non-empty.
// Allowed is the canonical import path (e.g. "log/slog", "go.uber.org/zap").
type LoggerConfig struct {
	Allowed string `yaml:"allowed"`
}

// FileSizeRule limits lines per file pattern or exact repo-relative path.
type FileSizeRule struct {
	Pattern string `yaml:"pattern"`
	File    string `yaml:"file"`
	Limit   int    `yaml:"limit"`
}

// MaxMethodsRule limits exported methods per file pattern.
type MaxMethodsRule struct {
	Pattern string `yaml:"pattern"`
	Limit   int    `yaml:"limit"`
}

// WiringConfig restricts where layer constructors may be called.
type WiringConfig struct {
	AllowedFiles []string `yaml:"allowed_files"`
}

// RuleConfig overrides severity for a rule id (e.g. rule5).
type RuleConfig struct {
	Severity string `yaml:"severity"`
}

// Severity is the effective enforcement level for a rule.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityOff
)

// DefaultSeverity is used when lint.rules does not override a rule.
var DefaultSeverity = map[string]Severity{
	"ports-required":        SeverityError,
	"ports-complete":        SeverityError,
	"handler-no-db":         SeverityError,
	"service-no-http":       SeverityError,
	"no-concrete-fields":    SeverityError,
	"layer-dependency":      SeverityError,
	"type-placement":        SeverityError,
	"file-size":             SeverityError,
	"cross-imports":         SeverityError,
	"constructor-injection": SeverityError,
	"test-files":            SeverityError,
	"no-panic":              SeverityError,
	"context-first":         SeverityError,
	"method-count":          SeverityError,
	"wiring-isolation":      SeverityError,
	"logger-no-other":       SeverityOff,
	"logger-no-fmt-print":   SeverityOff,
	"logger-no-init":        SeverityOff,
}

// RuleSeverity returns the effective severity for a rule id.
func (lc LintConfig) RuleSeverity(rule string) Severity {
	if rc, ok := lc.Rules[rule]; ok {
		return parseSeverity(rc.Severity)
	}
	if isLoggerRule(rule) {
		if strings.TrimSpace(lc.Logger.Allowed) == "" {
			return SeverityOff
		}
		return SeverityError
	}
	if d, ok := DefaultSeverity[rule]; ok {
		return d
	}
	return SeverityError
}

func isLoggerRule(rule string) bool {
	switch rule {
	case "logger-no-other", "logger-no-fmt-print", "logger-no-init":
		return true
	default:
		return false
	}
}

func parseSeverity(s string) Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "warning", "warn":
		return SeverityWarning
	case "off", "disable", "disabled", "none":
		return SeverityOff
	case "error", "":
		return SeverityError
	default:
		return SeverityError
	}
}

// Load reads portsmith.yaml from projectRoot. Missing file returns zero Config (no error).
func Load(projectRoot string) (Config, error) {
	path := filepath.Join(projectRoot, "portsmith.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("lintconfig: read %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("lintconfig: parse %s: %w", path, err)
	}
	return cfg, nil
}
