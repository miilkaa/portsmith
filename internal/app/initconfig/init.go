// Package initconfig implements the portsmith init command.
package initconfig

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stackChiSQLx = "chi-sqlx"
	stackGinGorm = "gin-gorm"

	loggerSkip    = "skip"
	loggerSlog    = "log/slog"
	loggerZap     = "go.uber.org/zap"
	loggerZerolog = "github.com/rs/zerolog"
	loggerLogrus  = "github.com/sirupsen/logrus"

	maxLinesSkip = 0

	maxMethodsSkip = 0

	wiringSkip    = "skip"
	wiringDefault = "default"
	wiringCustom  = "custom"
)

// Options configures init without the interactive wizard (tests and tooling).
type Options struct {
	// Dir is the project root directory. Empty means current working directory.
	Dir string
	// Force overwrites an existing portsmith.yaml.
	Force bool
	// Lang is "en", "ru", or empty to detect from the environment.
	Lang string
	// Answers, if non-nil, skips huh and writes YAML from these values.
	Answers *WizardAnswers
}

// WizardAnswers holds wizard selections; used when bypassing interactive mode.
type WizardAnswers struct {
	Stack           string
	LoggerImport    string // empty or canonical import (skip uses empty)
	MaxLinesLimit   int    // 0 = skip
	MaxMethodsLimit int    // 0 = skip
	WiringMode      string
	WiringFiles     string // comma-separated when WiringMode is wiringCustom
}

// Run parses flags and runs the interactive wizard in the current directory.
func Run(args []string) error {
	var force bool
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.BoolVar(&force, "force", false, "overwrite existing portsmith.yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments\n\nusage: portsmith init [--force]")
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	return RunWithOptions(Options{Dir: dir, Force: force})
}

// RunWithOptions runs the wizard or writes config from preset Answers.
func RunWithOptions(opts Options) error {
	dir := opts.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd: %w", err)
		}
	}

	outPath := filepath.Join(dir, "portsmith.yaml")
	if _, err := os.Stat(outPath); err == nil && !opts.Force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", outPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", outPath, err)
	}

	lang := opts.Lang
	if lang == "" {
		lang = DetectLang()
	}
	if lang != "ru" {
		lang = "en"
	}
	loc := localeFor(lang)

	var answers WizardAnswers
	if opts.Answers != nil {
		answers = *opts.Answers
	} else {
		if err := runWizard(&answers, loc); err != nil {
			return err
		}
	}

	module := ""
	if mod, err := readModuleFromGoMod(filepath.Join(dir, "go.mod")); err == nil {
		module = mod
	}

	data := buildPortsmithYAMLString(answers, module)
	if err := os.WriteFile(outPath, []byte(data), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	// Only print when running the interactive wizard (preset Answers is for tests/scripts).
	if opts.Answers == nil {
		fmt.Println(doneMessage(loc, outPath))
	}
	return nil
}
