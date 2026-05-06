package cli

import "fmt"

// PrintUsage prints the top-level portsmith usage.
func PrintUsage() {
	fmt.Print(`portsmith — Go Clean Architecture toolkit

Usage:
  portsmith init  [--force]
      Interactive wizard: writes portsmith.yaml in the current directory.
      Prompt language follows LC_ALL / LC_MESSAGES / LANG.
      --force   overwrite an existing portsmith.yaml

  portsmith gen   [--dry-run] [--all] [--scan-callers] [<pkg-dir>...]
      Generate ports.go for one or more packages.

  portsmith new   [--stack gin-gorm|chi-sqlx] <pkg-dir>
      Scaffold a new package with direct Gin/GORM or Chi/sqlx code.

  portsmith mock  [<pkg-dir>...]
      Generate mocks for all interfaces in ports.go via mockery.

  portsmith check [--stack gin-gorm|chi-sqlx] [<pkg-dir>...]
      Validate Clean Architecture rules.

  portsmith version
      Print the installed version.

  portsmith help
      Print this help message.
`)
}
