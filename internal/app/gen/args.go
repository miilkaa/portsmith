package gen

type genOptions struct {
	dryRun      bool
	scanCallers bool
	verbose     bool
	patterns    []string
}

func parseArgs(args []string) genOptions {
	var opts genOptions

	for _, arg := range args {
		switch arg {
		case "--dry-run":
			opts.dryRun = true
		case "--scan-callers":
			opts.scanCallers = true
		case "-v", "--verbose":
			opts.verbose = true
		default:
			opts.patterns = append(opts.patterns, arg)
		}
	}

	return opts
}
