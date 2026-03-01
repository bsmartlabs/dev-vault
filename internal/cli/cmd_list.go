package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"regexp"
	"text/tabwriter"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func runList(ctx commandContext, argv []string) int {
	var contains stringSliceFlag
	var nameRegex string
	var pathFilter string
	var typeFilter string
	var jsonOut bool

	return runParsedCommand(ctx, argv, commandSpec{
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
	}, func(parsed *parsedCommand) int {
		return newCommandRuntime(ctx, parsed).execute(func(_ *config.Loaded, service commandService) error {
			var re *regexp.Regexp
			if nameRegex != "" {
				compiled, err := regexp.Compile(nameRegex)
				if err != nil {
					return usageError(fmt.Errorf("invalid --name-regex: %w", err))
				}
				re = compiled
			}

			filtered, err := service.list(listQuery{
				NameContains: contains,
				NameRegex:    re,
				Path:         pathFilter,
				Type:         typeFilter,
			})
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(ctx.stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(filtered); err != nil {
					return outputError(err)
				}
				return nil
			}

			tw := tabwriter.NewWriter(ctx.stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tTYPE\tPATH\tID")
			for _, it := range filtered {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID)
			}
			if err := tw.Flush(); err != nil {
				return outputError(err)
			}
			return nil
		})
	})
}
