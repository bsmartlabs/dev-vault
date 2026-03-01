package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

var listCommandDef = commandDef{
	Name:    "list",
	Summary: "List project -dev secrets metadata",
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
			"It is not limited to entries present in .scw.json mapping.",
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

type listOptions struct {
	json         bool
	nameContains []string
	nameRegex    string
	path         string
	secretType   string
}

func parseListOptions(parsed *parsedCommand) listOptions {
	return listOptions{
		json:         parsed.Bool("json"),
		nameContains: parsed.Strings("name-contains"),
		nameRegex:    parsed.String("name-regex"),
		path:         parsed.String("path"),
		secretType:   parsed.String("type"),
	}
}

func runListParsed(ctx commandContext, parsed *parsedCommand) int {
	opts := parseListOptions(parsed)
	if err := rejectUnexpectedArgs(parsed, "list"); err != nil {
		return newCommandRuntime(ctx, parsed).writeStderrError(err)
	}
	return newCommandRuntime(ctx, parsed).execute(func(_ *config.Loaded, service secretsync.Service) error {
		var re *regexp.Regexp
		var selectedType secretprovider.SecretType

		if opts.nameRegex != "" {
			compiled, err := regexp.Compile(opts.nameRegex)
			if err != nil {
				return usageError(fmt.Errorf("invalid --name-regex: %w", err))
			}
			re = compiled
		}

		if opts.secretType != "" {
			parsedType, err := parseSecretType(opts.secretType)
			if err != nil {
				return usageError(fmt.Errorf("invalid --type: %w", err))
			}
			selectedType = parsedType
		}

		filtered, err := service.List(secretsync.ListQuery{
			NameContains: opts.nameContains,
			NameRegex:    re,
			Path:         opts.path,
			Type:         selectedType,
		})
		if err != nil {
			return err
		}

		if opts.json {
			enc := json.NewEncoder(ctx.stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(filtered); err != nil {
				return outputError(err)
			}
			return nil
		}

		if _, err := fmt.Fprintln(ctx.stdout, "NAME\tTYPE\tPATH\tID"); err != nil {
			return outputError(err)
		}
		for _, it := range filtered {
			if _, err := fmt.Fprintf(ctx.stdout, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID); err != nil {
				return outputError(err)
			}
		}
		return nil
	})
}
