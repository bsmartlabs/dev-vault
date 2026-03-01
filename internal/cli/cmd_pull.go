package cli

import (
	"flag"
	"fmt"
)

func runPull(ctx commandContext, argv []string) int {
	var all bool
	var overwrite bool
	parsed, parseErr := parseCommand(ctx, argv, commandSpec{
		name:           "pull",
		usage:          printPullUsage,
		localFlagSpecs: map[string]bool{"all": false, "overwrite": false},
		bindFlags: func(fs *flag.FlagSet, _ *string, _ *string) {
			fs.BoolVar(&all, "all", false, "Pull all mapping entries with mode pull|both (mode defaults to both)")
			fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
		},
	})
	if code, terminal := parseCommandExitCode(parseErr); terminal {
		return code
	}

	return newCommandRuntime(ctx, parsed).executeMapping(mappingCommandSpec{
		mode: "pull",
		all:  all,
		execute: func(service commandService, targets []mappingTarget) error {
			results, err := service.pull(targets, overwrite)
			if err != nil {
				return err
			}
			for _, item := range results {
				if _, err := fmt.Fprintf(ctx.stdout, "pulled %s -> %s (rev=%d type=%s)\n", item.Name, item.File, item.Revision, item.Type); err != nil {
					return outputError(err)
				}
			}
			return nil
		},
	})
}
