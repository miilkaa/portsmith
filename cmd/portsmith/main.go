// Command portsmith is the CLI entry point for the portsmith toolkit.
package main

import (
	"fmt"
	"os"

	"github.com/miilkaa/portsmith/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
