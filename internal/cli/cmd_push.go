package cli

import (
	"flag"
	"fmt"
)

func runPush(ctx commandContext, argv []string) int {
	var all bool
	var yes bool
	var disablePrevious bool
	var description string
	var createMissing bool

	parsed, exitCode := parseCommand(ctx, argv, commandSpec{
		name:  "push",
		usage: printPushUsage,
		localFlagSpecs: map[string]bool{
			"all":              false,
			"yes":              false,
			"disable-previous": false,
			"description":      true,
			"create-missing":   false,
		},
		bindFlags: func(fs *flag.FlagSet, _ *string, _ *string) {
			fs.BoolVar(&all, "all", false, "Push all mapping entries with mode push|both (mode defaults to both)")
			fs.BoolVar(&yes, "yes", false, "Confirm batch push (required when pushing more than one secret)")
			fs.BoolVar(&disablePrevious, "disable-previous", false, "Disable previous enabled version when creating a new version")
			fs.StringVar(&description, "description", "", "Description for the new version (optional)")
			fs.BoolVar(&createMissing, "create-missing", false, "Create missing secrets (requires mapping.type)")
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

	targets, err := selectMappingTargets(loaded.Cfg.Mapping, all, parsed.fs.Args(), "push")
	printConfigWarnings(ctx.stderr, loaded.Warnings)
	if err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}
	if len(targets) > 1 && !yes {
		err := usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}

	results, err := service.push(targets, pushOptions{
		Description:     description,
		DisablePrevious: disablePrevious,
		CreateMissing:   createMissing,
	})
	if err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}

	for _, item := range results {
		if _, err := fmt.Fprintf(ctx.stdout, "pushed %s (rev=%d)\n", item.Name, item.Revision); err != nil {
			fmt.Fprintln(ctx.stderr, err.Error())
			return exitCodeForError(outputError(err))
		}
	}
	return 0
}
