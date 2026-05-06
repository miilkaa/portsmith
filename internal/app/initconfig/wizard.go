package initconfig

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

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
