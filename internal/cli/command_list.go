package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

var listCommandDef = commandDef{
	Name:    "list",
	Summary: "List mapped -dev secrets metadata",
	Flags: []commandFlagDef{
		{Name: "json", Kind: commandFlagBool, Help: "Output JSON"},
		{Name: "name-contains", Kind: commandFlagStringSlice, ValueName: "<substring>", Help: "Substring filter (repeatable, AND semantics)"},
		{Name: "name-regex", Kind: commandFlagString, ValueName: "<regexp>", Help: "Go regexp to match secret names"},
		{Name: "path", Kind: commandFlagString, ValueName: "<path>", Help: "Exact Scaleway secret path to filter"},
		{Name: "type", Kind: commandFlagString, ValueName: "<type>", Help: fmt.Sprintf("One of: %s", strings.Join(secrettype.Names(), "|"))},
	},
	Doc: commandDoc{
		Synopsis: "dev-vault [--config <path>] [--profile <name>] list [options]",
		Description: []string{
			"Lists secrets in the configured Scaleway project/region.",
			"This command always filters to secret names ending with '-dev'.",
			"It never prints secret payloads, only metadata (name/type/path/id).",
		},
		Examples: []string{
			"dev-vault list",
			"dev-vault list --json",
			"dev-vault list --name-contains bweb --name-contains env",
			"dev-vault list --name-regex '^bweb-env-.*-dev$' --path / --type key_value",
		},
	},
	RunParsed: runListParsed,
}

func runList(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, listCommandDef)
}

func runListParsed(ctx commandContext, parsed *parsedCommand) int {
	return newCommandRuntime(ctx, parsed).execute(func(_ *config.Loaded, service secretsync.Service) error {
		var re *regexp.Regexp
		var selectedType secretprovider.SecretType

		nameRegex := parsed.String("name-regex")
		if nameRegex != "" {
			compiled, err := regexp.Compile(nameRegex)
			if err != nil {
				return usageError(fmt.Errorf("invalid --name-regex: %w", err))
			}
			re = compiled
		}

		typeFilter := parsed.String("type")
		if typeFilter != "" {
			parsedType, err := secretsync.ParseSecretType(typeFilter)
			if err != nil {
				return usageError(fmt.Errorf("invalid --type: %w", err))
			}
			selectedType = parsedType
		}

		filtered, err := service.List(secretsync.ListQuery{
			NameContains: parsed.Strings("name-contains"),
			NameRegex:    re,
			Path:         parsed.String("path"),
			Type:         selectedType,
		})
		if err != nil {
			return err
		}

		if parsed.Bool("json") {
			enc := json.NewEncoder(ctx.stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(filtered); err != nil {
				return outputError(err)
			}
			return nil
		}

		tw := tabwriter.NewWriter(ctx.stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "NAME\tTYPE\tPATH\tID")
		for _, it := range filtered {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID)
		}
		if err := tw.Flush(); err != nil {
			return outputError(err)
		}
		return nil
	})
}
