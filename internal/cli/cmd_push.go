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

	parsed, exitCode, err := parseCommand(ctx, argv, commandSpec{
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

	results, warnings, err := executePush(parsed.configPath, parsed.profileOverride, ctx.deps, all, parsed.fs.Args(), yes, pushOptions{
		Description:     description,
		DisablePrevious: disablePrevious,
		CreateMissing:   createMissing,
	})
	printConfigWarnings(ctx.stderr, warnings)
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
