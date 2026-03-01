package cli

import (
	"flag"
	"fmt"
)

func runPull(ctx commandContext, argv []string) int {
	var all bool
	var overwrite bool
	parsed, exitCode := parseCommand(ctx, argv, commandSpec{
		name:           "pull",
		usage:          printPullUsage,
		localFlagSpecs: map[string]bool{"all": false, "overwrite": false},
		bindFlags: func(fs *flag.FlagSet, _ *string, _ *string) {
			fs.BoolVar(&all, "all", false, "Pull all mapping entries with mode pull|both (mode defaults to both)")
			fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
		},
	})
	if exitCode >= 0 {
		return exitCode
	}

	loaded, api, err := loadAndOpenAPI(parsed.configPath, parsed.profileOverride, ctx.deps)
	if err != nil {
		runErr := runtimeError(err)
		fmt.Fprintln(ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}
	service := newCommandService(loaded, api, ctx.deps)

	targets, err := selectMappingTargets(loaded.Cfg.Mapping, all, parsed.fs.Args(), "pull")
	printConfigWarnings(ctx.stderr, loaded.Warnings)
	if err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}

	results, err := service.pull(targets, overwrite)
	if err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}

	for _, item := range results {
		if _, err := fmt.Fprintf(ctx.stdout, "pulled %s -> %s (rev=%d type=%s)\n", item.Name, item.File, item.Revision, item.Type); err != nil {
			fmt.Fprintln(ctx.stderr, err.Error())
			return exitCodeForError(outputError(err))
		}
	}
	return 0
}
