package check

type checkOptions struct {
	patterns []string
}

func parseArgs(args []string) checkOptions {
	var opts checkOptions

	for _, arg := range args {
		opts.patterns = append(opts.patterns, arg)
	}

	return opts
}
