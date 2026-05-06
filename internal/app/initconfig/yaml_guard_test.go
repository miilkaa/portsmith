// Package initconfig_yamlguard_test contains a reflection-based guard that ensures
// every field of project.LintConfig appears in the generated portsmith.yaml.
//
// When a new field is added to LintConfig but BuildPortsmithYAMLString is not
// updated, this test fails — keeping the wizard output in sync automatically.
package initconfig_test

import (
	"reflect"
	"strings"
	"testing"

	initconfig "github.com/miilkaa/portsmith/internal/app/initconfig"
	"github.com/miilkaa/portsmith/internal/project"
)

// TestBuildPortsmithYAML_containsAllLintConfigFields reflects on project.LintConfig,
// collects every yaml struct tag, and verifies each appears in the generated YAML output.
//
// HOW TO FIX: if this test fails after adding a new field to LintConfig, update
// BuildPortsmithYAMLString (init.go) so the new yaml key appears in the output,
// then re-run the test.
func TestBuildPortsmithYAML_containsAllLintConfigFields(t *testing.T) {
	// Use a "full" answer set so that active sections are included too.
	answers := initconfig.WizardAnswers{
		Stack:           "chi-sqlx",
		LoggerImport:    "log/slog",
		MaxLinesLimit:   300,
		MaxMethodsLimit: 15,
		WiringMode:      "default",
	}
	output := initconfig.BuildPortsmithYAMLString(answers, "github.com/example/myapp")

	yamlTags := collectYAMLTags(reflect.TypeOf(project.LintConfig{}))
	for _, tag := range yamlTags {
		if tag == "" || tag == "-" {
			continue
		}
		if !strings.Contains(output, tag) {
			t.Errorf(
				"generated portsmith.yaml is missing yaml key %q\n\n"+
					"Fix: add this key (with documentation) to BuildPortsmithYAMLString in init.go",
				tag,
			)
		}
	}
}

// TestBuildPortsmithYAML_rootStackField checks the top-level "stack" key is present.
func TestBuildPortsmithYAML_rootStackField(t *testing.T) {
	answers := initconfig.WizardAnswers{Stack: "gin-gorm"}
	output := initconfig.BuildPortsmithYAMLString(answers, "")
	if !strings.Contains(output, "stack:") {
		t.Error("generated YAML is missing top-level 'stack' key")
	}
}

// collectYAMLTags returns the yaml struct tag values (first comma token, no options)
// for all fields in t recursively (only named struct types, no infinite loops).
func collectYAMLTags(t reflect.Type) []string {
	return collectYAMLTagsInner(t, make(map[reflect.Type]bool))
}

func collectYAMLTagsInner(t reflect.Type, seen map[reflect.Type]bool) []string {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	if seen[t] {
		return nil
	}
	seen[t] = true

	var tags []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		raw := field.Tag.Get("yaml")
		if raw != "" && raw != "-" {
			// Take only the name part before the first comma.
			name := strings.SplitN(raw, ",", 2)[0]
			if name != "" && name != "-" {
				tags = append(tags, name)
			}
		}
		// Recurse into nested structs.
		tags = append(tags, collectYAMLTagsInner(field.Type, seen)...)
	}
	return tags
}
