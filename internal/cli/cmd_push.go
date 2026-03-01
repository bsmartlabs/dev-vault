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

	parsed, parseErr := parseCommand(ctx, argv, commandSpec{
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
	if code, terminal := parseCommandExitCode(parseErr); terminal {
		return code
	}

	return newCommandRuntime(ctx, parsed).executeMapping(mappingCommandSpec{
		mode: "push",
		all:  all,
		preflight: func(targets []mappingTarget) error {
			if len(targets) > 1 && !yes {
				return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
			}
			return nil
		},
		execute: func(service commandService, targets []mappingTarget) error {
			results, err := service.push(targets, pushOptions{
				Description:     description,
				DisablePrevious: disablePrevious,
				CreateMissing:   createMissing,
			})
			if err != nil {
				return err
			}
			for _, item := range results {
				if _, err := fmt.Fprintf(ctx.stdout, "pushed %s (rev=%d)\n", item.Name, item.Revision); err != nil {
					return outputError(err)
				}
			}
			return nil
		},
	})
}
