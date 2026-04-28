// Package initcmd implements the "portsmith init" command — an interactive
// wizard that writes portsmith.yaml in the current project directory.
//
//	portsmith init [--force]
package initcmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
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
	Stack          string
	LoggerImport   string // empty or canonical import (skip uses empty)
	MaxLinesLimit  int  // 0 = skip
	MaxMethodsLimit int // 0 = skip
	WiringMode     string
	WiringFiles    string // comma-separated when WiringMode is wiringCustom
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

// DetectLang returns "ru" if LC_ALL/LC_MESSAGES/LANG suggests Russian, else "en".
// LC_ALL is checked first (POSIX: it overrides LANG).
func DetectLang() string {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(env); v != "" {
			lower := strings.ToLower(v)
			if strings.HasPrefix(lower, "ru") {
				return "ru"
			}
			return "en"
		}
	}
	return "en"
}

type localeBundle struct {
	stackTitle       string
	stackDesc        string
	stackChi         string
	stackGin         string
	loggerTitle      string
	loggerDesc       string
	loggerOptSlog    string
	loggerOptZap     string
	loggerOptZerolog string
	loggerOptLogrus  string
	loggerOptSkip    string
	maxLinesTitle    string
	maxLinesDesc     string
	maxLines150      string
	maxLines300      string
	maxLines500      string
	maxLinesSkip     string
	maxMethodsTitle  string
	maxMethodsDesc   string
	maxMethods10     string
	maxMethods15     string
	maxMethods20     string
	maxMethodsSkip   string
	wiringTitle      string
	wiringDesc       string
	wiringDefault    string
	wiringCustom     string
	wiringSkip       string
	wiringInputTitle string
	wiringInputDesc  string
	wiringPlaceholder string
	formTitle        string
	formDesc         string
}

func localeFor(lang string) localeBundle {
	if lang == "ru" {
		return localeBundle{
			stackTitle:  "Стек",
			stackDesc:   "Выберите стек Portsmith (влияет на шаблоны new/gen/check).",
			stackChi:    "Chi + sqlx (chi-sqlx)",
			stackGin:    "Gin + GORM (gin-gorm)",
			loggerTitle: "Логгер",
			loggerDesc:  "Разрешённый пакет логирования для правил logger-* (пусто — правила выключены).",
			loggerOptSlog:    "log/slog (стандартная библиотека)",
			loggerOptZap:     "go.uber.org/zap",
			loggerOptZerolog: "github.com/rs/zerolog",
			loggerOptLogrus:  "github.com/sirupsen/logrus",
			loggerOptSkip:    "Пропустить (не задавать lint.logger)",
			maxLinesTitle: "Лимит строк в файле",
			maxLinesDesc:  "Правило file-size: максимум строк на файл по шаблону **/*.go.",
			maxLines150:   "150 строк",
			maxLines300:   "300 строк",
			maxLines500:   "500 строк",
			maxLinesSkip:  "Пропустить (закомментировать в YAML)",
			maxMethodsTitle: "Лимит методов на тип",
			maxMethodsDesc:  "Правило method-count: максимум экспортируемых методов на файл.",
			maxMethods10:    "10 методов",
			maxMethods15:    "15 методов",
			maxMethods20:    "20 методов",
			maxMethodsSkip:  "Пропустить (закомментировать в YAML)",
			wiringTitle:  "Файлы wiring",
			wiringDesc:   "Где разрешено вызывать конструкторы слоёв (wiring-isolation).",
			wiringDefault: "По умолчанию: wire.go, app.go",
			wiringCustom:  "Задать вручную (через запятую)",
			wiringSkip:    "Пропустить (закомментировать в YAML)",
			wiringInputTitle: "Имена файлов wiring",
			wiringInputDesc:  "Список через запятую, например: wire.go, cmd/wire.go",
			wiringPlaceholder: "wire.go, app.go",
			formTitle: "portsmith init",
			formDesc:  "Интерактивная настройка portsmith.yaml",
		}
	}
	return localeBundle{
		stackTitle:  "Stack",
		stackDesc:   "Portsmith stack (affects new/gen/check templates).",
		stackChi:    "Chi + sqlx (chi-sqlx)",
		stackGin:    "Gin + GORM (gin-gorm)",
		loggerTitle: "Logger",
		loggerDesc:  "Allowed logging import for logger-* lint rules (empty disables those rules).",
		loggerOptSlog:    "log/slog (stdlib)",
		loggerOptZap:     "go.uber.org/zap",
		loggerOptZerolog: "github.com/rs/zerolog",
		loggerOptLogrus:  "github.com/sirupsen/logrus",
		loggerOptSkip:    "Skip (omit lint.logger)",
		maxLinesTitle: "Max lines per file",
		maxLinesDesc:  "file-size rule: max lines per file matching **/*.go.",
		maxLines150:   "150 lines",
		maxLines300:   "300 lines",
		maxLines500:   "500 lines",
		maxLinesSkip:  "Skip (commented in YAML)",
		maxMethodsTitle: "Max methods per type",
		maxMethodsDesc:  "method-count rule: max exported methods per file.",
		maxMethods10:    "10 methods",
		maxMethods15:    "15 methods",
		maxMethods20:    "20 methods",
		maxMethodsSkip:  "Skip (commented in YAML)",
		wiringTitle:  "Wiring files",
		wiringDesc:   "Where layer constructors may be called (wiring-isolation).",
		wiringDefault: "Default: wire.go, app.go",
		wiringCustom:  "Custom (comma-separated)",
		wiringSkip:    "Skip (commented in YAML)",
		wiringInputTitle: "Wiring file names",
		wiringInputDesc:  "Comma-separated, e.g. wire.go, cmd/wire.go",
		wiringPlaceholder: "wire.go, app.go",
		formTitle: "portsmith init",
		formDesc:  "Interactive portsmith.yaml setup",
	}
}

func doneMessage(loc localeBundle, path string) string {
	if strings.Contains(loc.formDesc, "Interactive") {
		return fmt.Sprintf("Wrote %s", path)
	}
	return fmt.Sprintf("Создан файл %s", path)
}

// BuildPortsmithYAMLString renders the full portsmith.yaml content from wizard answers.
// The active selections are uncommented; every other option appears commented with
// full documentation so the user can see all available knobs.
// Exported so tests and tooling can inspect the output without running the wizard.
func BuildPortsmithYAMLString(a WizardAnswers, module string) string {
	return buildPortsmithYAMLString(a, module)
}

func runWizard(a *WizardAnswers, loc localeBundle) error {
	var loggerChoice string
	var maxLinesChoice int
	var maxMethodsChoice int
	var wiringFiles string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(loc.stackTitle).
				Description(loc.stackDesc).
				Options(
					huh.NewOption(loc.stackChi, stackChiSQLx),
					huh.NewOption(loc.stackGin, stackGinGorm),
				).
				Value(&a.Stack),

			huh.NewSelect[string]().
				Title(loc.loggerTitle).
				Description(loc.loggerDesc).
				Options(
					huh.NewOption(loc.loggerOptSlog, loggerSlog),
					huh.NewOption(loc.loggerOptZap, loggerZap),
					huh.NewOption(loc.loggerOptZerolog, loggerZerolog),
					huh.NewOption(loc.loggerOptLogrus, loggerLogrus),
					huh.NewOption(loc.loggerOptSkip, loggerSkip),
				).
				Value(&loggerChoice),

			huh.NewSelect[int]().
				Title(loc.maxLinesTitle).
				Description(loc.maxLinesDesc).
				Options(
					huh.NewOption(loc.maxLines150, 150),
					huh.NewOption(loc.maxLines300, 300),
					huh.NewOption(loc.maxLines500, 500),
					huh.NewOption(loc.maxLinesSkip, maxLinesSkip),
				).
				Value(&maxLinesChoice),

			huh.NewSelect[int]().
				Title(loc.maxMethodsTitle).
				Description(loc.maxMethodsDesc).
				Options(
					huh.NewOption(loc.maxMethods10, 10),
					huh.NewOption(loc.maxMethods15, 15),
					huh.NewOption(loc.maxMethods20, 20),
					huh.NewOption(loc.maxMethodsSkip, maxMethodsSkip),
				).
				Value(&maxMethodsChoice),

			huh.NewSelect[string]().
				Title(loc.wiringTitle).
				Description(loc.wiringDesc).
				Options(
					huh.NewOption(loc.wiringDefault, wiringDefault),
					huh.NewOption(loc.wiringCustom, wiringCustom),
					huh.NewOption(loc.wiringSkip, wiringSkip),
				).
				Value(&a.WiringMode),
		).Title(loc.formTitle).Description(loc.formDesc),

		huh.NewGroup(
			huh.NewInput().
				Title(loc.wiringInputTitle).
				Description(loc.wiringInputDesc).
				Placeholder(loc.wiringPlaceholder).
				Value(&wiringFiles).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						if loc.wiringInputTitle == "Wiring file names" {
							return fmt.Errorf("enter at least one file name")
						}
						return fmt.Errorf("укажите хотя бы одно имя файла")
					}
					return nil
				}),
		).WithHideFunc(func() bool {
			return a.WiringMode != wiringCustom
		}),
	)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return fmt.Errorf("cancelled")
		}
		return err
	}

	if loggerChoice != loggerSkip {
		a.LoggerImport = loggerChoice
	}
	a.MaxLinesLimit = maxLinesChoice
	a.MaxMethodsLimit = maxMethodsChoice
	if a.WiringMode == wiringCustom {
		a.WiringFiles = wiringFiles
	}
	return nil
}

func buildPortsmithYAMLString(a WizardAnswers, module string) string {
	var b strings.Builder
	sep := func() { b.WriteString("  # " + strings.Repeat("─", 74) + "\n") }

	// ── header ───────────────────────────────────────────────────────────────
	b.WriteString("# Generated by portsmith init\n")
	b.WriteString("# https://github.com/miilkaa/portsmith\n")
	if module != "" {
		fmt.Fprintf(&b, "# module: %s\n", module)
	}
	b.WriteString("#\n")
	b.WriteString("# Active settings are uncommented. Everything else is commented with full\n")
	b.WriteString("# documentation — uncomment and adjust to enable.\n")
	b.WriteString("\n")

	// ── stack ─────────────────────────────────────────────────────────────────
	b.WriteString("# stack — framework stack used by portsmith new / gen / check templates.\n")
	b.WriteString("#   chi-sqlx  — Go Chi router (github.com/go-chi/chi) + jmoiron/sqlx (raw SQL)\n")
	b.WriteString("#   gin-gorm  — Gin web framework + GORM ORM (code-first migrations)\n")
	fmt.Fprintf(&b, "stack: %s\n\n", a.Stack)

	// ── lint block ────────────────────────────────────────────────────────────
	b.WriteString("lint:\n")

	// ── logger ────────────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # logger.allowed — canonical import path of the ONE logger permitted in this\n")
	b.WriteString("  # project. When set, three rules activate automatically:\n")
	b.WriteString("  #   • logger-no-other     — only the specified logger may be imported\n")
	b.WriteString("  #   • logger-no-fmt-print  — fmt.Print* / fmt.Println are forbidden\n")
	b.WriteString("  #   • logger-no-init       — package-level logger variables are forbidden\n")
	b.WriteString("  #\n")
	b.WriteString("  # Common values:\n")
	b.WriteString("  #   log/slog                   — Go 1.21+ structured logger (stdlib)\n")
	b.WriteString("  #   go.uber.org/zap            — high-performance structured logger\n")
	b.WriteString("  #   github.com/rs/zerolog      — zero-allocation JSON logger\n")
	b.WriteString("  #   github.com/sirupsen/logrus — feature-rich structured logger\n")
	b.WriteString("  #\n")
	b.WriteString("  # Remove or leave empty to disable all logger-* rules entirely.\n")
	sep()
	if strings.TrimSpace(a.LoggerImport) != "" {
		b.WriteString("  logger:\n")
		fmt.Fprintf(&b, "    allowed: %s\n\n", a.LoggerImport)
	} else {
		b.WriteString("  # logger:\n")
		b.WriteString("  #   allowed: log/slog\n\n")
	}

	// ── max_lines ─────────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # max_lines — file-size rule.\n")
	b.WriteString("  # Limits the number of lines per file for any glob pattern or exact path.\n")
	b.WriteString("  # Multiple entries may be listed; first match wins.\n")
	b.WriteString("  #\n")
	b.WriteString("  # Fields:\n")
	b.WriteString("  #   pattern  — glob applied to repo-relative paths (e.g. \"**/*.go\")\n")
	b.WriteString("  #   file     — exact repo-relative path (alternative to pattern)\n")
	b.WriteString("  #   limit    — maximum number of lines (including blank lines and comments)\n")
	b.WriteString("  #\n")
	b.WriteString("  # Tip: downgrade to a warning in the rules section instead of disabling.\n")
	sep()
	if a.MaxLinesLimit > 0 {
		b.WriteString("  max_lines:\n")
		b.WriteString("    - pattern: \"**/*.go\"\n")
		fmt.Fprintf(&b, "      limit: %d\n\n", a.MaxLinesLimit)
	} else {
		b.WriteString("  # max_lines:\n")
		b.WriteString("  #   - pattern: \"**/*.go\"\n")
		b.WriteString("  #     limit: 300\n\n")
	}

	// ── max_methods ───────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # max_methods — method-count rule.\n")
	b.WriteString("  # Limits the number of exported methods declared in a single file.\n")
	b.WriteString("  # Helps keep service and repository files focused and small.\n")
	b.WriteString("  #\n")
	b.WriteString("  # Fields:\n")
	b.WriteString("  #   pattern  — glob applied to repo-relative paths\n")
	b.WriteString("  #   limit    — maximum number of exported methods per file\n")
	sep()
	if a.MaxMethodsLimit > 0 {
		b.WriteString("  max_methods:\n")
		b.WriteString("    - pattern: \"**/*.go\"\n")
		fmt.Fprintf(&b, "      limit: %d\n\n", a.MaxMethodsLimit)
	} else {
		b.WriteString("  # max_methods:\n")
		b.WriteString("  #   - pattern: \"**/*.go\"\n")
		b.WriteString("  #     limit: 15\n\n")
	}

	// ── wiring ────────────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # wiring.allowed_files — wiring-isolation rule.\n")
	b.WriteString("  # Lists the ONLY files where New*Repository / New*Service / New*Handler\n")
	b.WriteString("  # constructors may be called. Calls from any other file are flagged.\n")
	b.WriteString("  # This enforces that all dependency wiring stays in one place.\n")
	b.WriteString("  #\n")
	b.WriteString("  # Typical values: wire.go, app.go, internal/app/wire.go, cmd/server/main.go\n")
	sep()
	switch a.WiringMode {
	case wiringDefault:
		b.WriteString("  wiring:\n")
		b.WriteString("    allowed_files:\n")
		b.WriteString("      - wire.go\n")
		b.WriteString("      - app.go\n\n")
	case wiringCustom:
		files := parseCommaList(a.WiringFiles)
		if len(files) > 0 {
			b.WriteString("  wiring:\n")
			b.WriteString("    allowed_files:\n")
			for _, f := range files {
				fmt.Fprintf(&b, "      - %s\n", f)
			}
			b.WriteString("\n")
		} else {
			b.WriteString("  # wiring:\n")
			b.WriteString("  #   allowed_files:\n")
			b.WriteString("  #     - wire.go\n")
			b.WriteString("  #     - app.go\n\n")
		}
	default:
		b.WriteString("  # wiring:\n")
		b.WriteString("  #   allowed_files:\n")
		b.WriteString("  #     - wire.go\n")
		b.WriteString("  #     - app.go\n\n")
	}

	// ── allowed_cross_imports ─────────────────────────────────────────────────
	sep()
	b.WriteString("  # allowed_cross_imports — cross-imports rule.\n")
	b.WriteString("  # By default no internal package may import another (strict domain isolation).\n")
	b.WriteString("  # List explicit allowances here when one domain legitimately calls another.\n")
	b.WriteString("  #\n")
	b.WriteString("  # Format:\n")
	b.WriteString("  #   <importer-package>:   ← last path segment of the importing package\n")
	b.WriteString("  #     - <imported-package> ← last path segment of the imported package\n")
	b.WriteString("  #\n")
	b.WriteString("  # Example: orders may import users and catalog; campaigns may import contacts:\n")
	sep()
	b.WriteString("  # allowed_cross_imports:\n")
	b.WriteString("  #   orders:\n")
	b.WriteString("  #     - users\n")
	b.WriteString("  #     - catalog\n")
	b.WriteString("  #   campaigns:\n")
	b.WriteString("  #     - contacts\n\n")

	// ── call_patterns ─────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # call_patterns — call-pattern rule.\n")
	b.WriteString("  # Forbids specific three-segment call patterns (recv.field.method) inside\n")
	b.WriteString("  # handler or service files. Use * as a wildcard for any identifier segment.\n")
	b.WriteString("  # The rule is DISABLED entirely when both not_allowed lists are empty.\n")
	b.WriteString("  #\n")
	b.WriteString("  # Fields:\n")
	b.WriteString("  #   handler.not_allowed — patterns forbidden in handler files\n")
	b.WriteString("  #   handler.allowed     — patterns shown as a hint when not_allowed fires\n")
	b.WriteString("  #   service.not_allowed — patterns forbidden in service files\n")
	b.WriteString("  #   service.allowed     — patterns shown as a hint when not_allowed fires\n")
	b.WriteString("  #\n")
	b.WriteString("  # Example: prevent handlers from calling DB methods directly:\n")
	sep()
	b.WriteString("  # call_patterns:\n")
	b.WriteString("  #   handler:\n")
	b.WriteString("  #     not_allowed:\n")
	b.WriteString("  #       - \"*.*.db\"\n")
	b.WriteString("  #       - \"*.*.query\"\n")
	b.WriteString("  #       - \"*.*.exec\"\n")
	b.WriteString("  #     allowed:\n")
	b.WriteString("  #       - \"h.service.*\"\n")
	b.WriteString("  #   service:\n")
	b.WriteString("  #     not_allowed:\n")
	b.WriteString("  #       - \"*.*.Render\"\n")
	b.WriteString("  #     allowed:\n")
	b.WriteString("  #       - \"s.repo.*\"\n\n")

	// ── rules ─────────────────────────────────────────────────────────────────
	sep()
	b.WriteString("  # rules — per-rule severity overrides.\n")
	b.WriteString("  # Values: error (default) | warning (print, don't fail) | off (disable)\n")
	b.WriteString("  #\n")
	b.WriteString("  # Rule ID             Default  What it enforces\n")
	b.WriteString("  # ─────────────────────────────────────────────────────────────────────\n")
	b.WriteString("  # ports-required       error    ports.go must exist when H+S+R layout is present\n")
	b.WriteString("  # ports-complete       error    ports.go must declare interfaces for all used methods\n")
	b.WriteString("  # handler-no-db        error    handler must not import database drivers directly\n")
	b.WriteString("  # service-no-http      error    service must not import HTTP/router packages\n")
	b.WriteString("  # no-concrete-fields   error    struct fields must use port interfaces, not concrete types\n")
	b.WriteString("  # layer-dependency     error    handler→service→repository direction only; no skipping\n")
	b.WriteString("  # type-placement       error    exported types must live in the correct layer file\n")
	b.WriteString("  # file-size            error    enforce max_lines limit (requires max_lines above)\n")
	b.WriteString("  # cross-imports        error    enforce allowed_cross_imports list\n")
	b.WriteString("  # constructor-injection error   constructors must accept interfaces, not concrete types\n")
	b.WriteString("  # test-files           error    test files must be present alongside source files\n")
	b.WriteString("  # no-panic             error    panic() is not allowed in service/repository files\n")
	b.WriteString("  # context-first        error    context.Context must be first parameter on exported methods\n")
	b.WriteString("  # method-count         error    enforce max_methods limit (requires max_methods above)\n")
	b.WriteString("  # wiring-isolation     error    constructors may only be called in wiring allowed_files\n")
	b.WriteString("  # logger-no-other      off      only the allowed logger may be imported\n")
	b.WriteString("  # logger-no-fmt-print  off      fmt.Print* is forbidden when logger is configured\n")
	b.WriteString("  # logger-no-init       off      package-level logger variables are forbidden\n")
	b.WriteString("  # call-pattern         error    enforce call_patterns rules (requires call_patterns above)\n")
	sep()
	b.WriteString("  # rules:\n")
	b.WriteString("  #   file-size:      warning\n")
	b.WriteString("  #   no-panic:       warning\n")
	b.WriteString("  #   test-files:     warning\n")
	b.WriteString("  #   context-first:  warning\n")
	b.WriteString("  #   method-count:   warning\n")

	return b.String()
}

func parseCommaList(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readModuleFromGoMod(goModPath string) (string, error) {
	f, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}
