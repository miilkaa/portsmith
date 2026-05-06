package initconfig

import (
	"fmt"
	"os"
	"strings"
)

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
	stackTitle        string
	stackDesc         string
	stackChi          string
	stackGin          string
	loggerTitle       string
	loggerDesc        string
	loggerOptSlog     string
	loggerOptZap      string
	loggerOptZerolog  string
	loggerOptLogrus   string
	loggerOptSkip     string
	maxLinesTitle     string
	maxLinesDesc      string
	maxLines150       string
	maxLines300       string
	maxLines500       string
	maxLinesSkip      string
	maxMethodsTitle   string
	maxMethodsDesc    string
	maxMethods10      string
	maxMethods15      string
	maxMethods20      string
	maxMethodsSkip    string
	wiringTitle       string
	wiringDesc        string
	wiringDefault     string
	wiringCustom      string
	wiringSkip        string
	wiringInputTitle  string
	wiringInputDesc   string
	wiringPlaceholder string
	formTitle         string
	formDesc          string
}

func localeFor(lang string) localeBundle {
	if lang == "ru" {
		return localeBundle{
			stackTitle:        "Стек",
			stackDesc:         "Выберите стек Portsmith (влияет на шаблоны new/gen/check).",
			stackChi:          "Chi + sqlx (chi-sqlx)",
			stackGin:          "Gin + GORM (gin-gorm)",
			loggerTitle:       "Логгер",
			loggerDesc:        "Разрешённый пакет логирования для правил logger-* (пусто — правила выключены).",
			loggerOptSlog:     "log/slog (стандартная библиотека)",
			loggerOptZap:      "go.uber.org/zap",
			loggerOptZerolog:  "github.com/rs/zerolog",
			loggerOptLogrus:   "github.com/sirupsen/logrus",
			loggerOptSkip:     "Пропустить (не задавать lint.logger)",
			maxLinesTitle:     "Лимит строк в файле",
			maxLinesDesc:      "Правило file-size: максимум строк на файл по шаблону **/*.go.",
			maxLines150:       "150 строк",
			maxLines300:       "300 строк",
			maxLines500:       "500 строк",
			maxLinesSkip:      "Пропустить (закомментировать в YAML)",
			maxMethodsTitle:   "Лимит методов на тип",
			maxMethodsDesc:    "Правило method-count: максимум экспортируемых методов на файл.",
			maxMethods10:      "10 методов",
			maxMethods15:      "15 методов",
			maxMethods20:      "20 методов",
			maxMethodsSkip:    "Пропустить (закомментировать в YAML)",
			wiringTitle:       "Файлы wiring",
			wiringDesc:        "Где разрешено вызывать конструкторы слоёв (wiring-isolation).",
			wiringDefault:     "По умолчанию: wire.go, app.go",
			wiringCustom:      "Задать вручную (через запятую)",
			wiringSkip:        "Пропустить (закомментировать в YAML)",
			wiringInputTitle:  "Имена файлов wiring",
			wiringInputDesc:   "Список через запятую, например: wire.go, cmd/wire.go",
			wiringPlaceholder: "wire.go, app.go",
			formTitle:         "portsmith init",
			formDesc:          "Интерактивная настройка portsmith.yaml",
		}
	}
	return localeBundle{
		stackTitle:        "Stack",
		stackDesc:         "Portsmith stack (affects new/gen/check templates).",
		stackChi:          "Chi + sqlx (chi-sqlx)",
		stackGin:          "Gin + GORM (gin-gorm)",
		loggerTitle:       "Logger",
		loggerDesc:        "Allowed logging import for logger-* lint rules (empty disables those rules).",
		loggerOptSlog:     "log/slog (stdlib)",
		loggerOptZap:      "go.uber.org/zap",
		loggerOptZerolog:  "github.com/rs/zerolog",
		loggerOptLogrus:   "github.com/sirupsen/logrus",
		loggerOptSkip:     "Skip (omit lint.logger)",
		maxLinesTitle:     "Max lines per file",
		maxLinesDesc:      "file-size rule: max lines per file matching **/*.go.",
		maxLines150:       "150 lines",
		maxLines300:       "300 lines",
		maxLines500:       "500 lines",
		maxLinesSkip:      "Skip (commented in YAML)",
		maxMethodsTitle:   "Max methods per type",
		maxMethodsDesc:    "method-count rule: max exported methods per file.",
		maxMethods10:      "10 methods",
		maxMethods15:      "15 methods",
		maxMethods20:      "20 methods",
		maxMethodsSkip:    "Skip (commented in YAML)",
		wiringTitle:       "Wiring files",
		wiringDesc:        "Where layer constructors may be called (wiring-isolation).",
		wiringDefault:     "Default: wire.go, app.go",
		wiringCustom:      "Custom (comma-separated)",
		wiringSkip:        "Skip (commented in YAML)",
		wiringInputTitle:  "Wiring file names",
		wiringInputDesc:   "Comma-separated, e.g. wire.go, cmd/wire.go",
		wiringPlaceholder: "wire.go, app.go",
		formTitle:         "portsmith init",
		formDesc:          "Interactive portsmith.yaml setup",
	}
}

func doneMessage(loc localeBundle, path string) string {
	if strings.Contains(loc.formDesc, "Interactive") {
		return fmt.Sprintf("Wrote %s", path)
	}
	return fmt.Sprintf("Создан файл %s", path)
}
