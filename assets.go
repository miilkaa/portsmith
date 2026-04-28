// Package portsmith provides embedded assets used by the portsmith project.
//
// This root-level package exists solely to expose the examples/ directory
// via go:embed, since embed paths cannot contain "..".
package portsmith

import "embed"

// ExamplesFS contains the embedded reference packages (clean_package_example_en / _ru).
//
//go:embed examples
var ExamplesFS embed.FS
