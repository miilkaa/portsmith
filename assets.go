// Package portsmith provides embedded assets used by the portsmith CLI.
//
// This root-level package exists solely to expose the examples/ directory
// via go:embed, since embed paths cannot contain "..".
// The binary reads ExamplesFS at runtime to copy reference examples into
// newly initialised projects.
package portsmith

import "embed"

// ExamplesFS contains the embedded example packages.
// Used by "portsmith init" to seed internal/ with reference code.
//
//go:embed examples
var ExamplesFS embed.FS
