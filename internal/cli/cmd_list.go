package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"text/tabwriter"
)

func runList(ctx commandContext, argv []string) int {
	var contains stringSliceFlag
	var nameRegex string
	var pathFilter string
	var typeFilter string
	var jsonOut bool

	parsed, exitCode := parseCommand(ctx, argv, commandSpec{
		name:  "list",
		usage: printListUsage,
		localFlagSpecs: map[string]bool{
			"name-contains": true,
			"name-regex":    true,
			"path":          true,
			"type":          true,
			"json":          false,
		},
		bindFlags: func(fs *flag.FlagSet, _ *string, _ *string) {
			fs.Var(&contains, "name-contains", "Substring filter (repeatable)")
			fs.StringVar(&nameRegex, "name-regex", "", "Go regexp to match secret names")
			fs.StringVar(&pathFilter, "path", "", "Exact Scaleway secret path to filter")
			fs.StringVar(&typeFilter, "type", "", "Secret type filter")
			fs.BoolVar(&jsonOut, "json", false, "Output JSON")
		},
	})
	if exitCode >= 0 {
		return exitCode
	}

	var re *regexp.Regexp
	var err error
	if nameRegex != "" {
		compiled, err := regexp.Compile(nameRegex)
		if err != nil {
			err = usageError(fmt.Errorf("invalid --name-regex: %w", err))
			fmt.Fprintln(ctx.stderr, err.Error())
			return exitCodeForError(err)
		}
		re = compiled
	}

	loaded, api, err := loadAndOpenAPI(parsed.configPath, parsed.profileOverride, ctx.deps)
	if err != nil {
		runErr := runtimeError(err)
		fmt.Fprintln(ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}
	service := newCommandService(loaded, api, ctx.deps)

	filtered, err := service.list(listQuery{
		NameContains: contains,
		NameRegex:    re,
		Path:         pathFilter,
		Type:         typeFilter,
	})
	printConfigWarnings(ctx.stderr, loaded.Warnings)
	if err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}

	if jsonOut {
		enc := json.NewEncoder(ctx.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(filtered); err != nil {
			fmt.Fprintf(ctx.stderr, "encode json: %v\n", err)
			return exitCodeForError(outputError(err))
		}
		return 0
	}

	tw := tabwriter.NewWriter(ctx.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tPATH\tID")
	for _, it := range filtered {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID)
	}
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(outputError(err))
	}
	return 0
}
