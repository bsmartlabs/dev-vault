package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

const (
	listFlagJSON         = "json"
	listFlagNameContains = "name-contains"
	listFlagNameRegex    = "name-regex"
	listFlagPath         = "path"
	listFlagType         = "type"
)

var listCommandDef = commandDef{
	Name:    "list",
	Summary: "List project -dev secrets metadata",
	Flags: []commandFlagDef{
		{Name: listFlagJSON, Kind: commandFlagBool, Help: "Output JSON"},
		{Name: listFlagNameContains, Kind: commandFlagStringSlice, ValueName: "<substring>", Help: "Substring filter (repeatable, AND semantics)"},
		{Name: listFlagNameRegex, Kind: commandFlagString, ValueName: "<regexp>", Help: "Go regexp to match secret names"},
		{Name: listFlagPath, Kind: commandFlagString, ValueName: "<path>", Help: "Exact Scaleway secret path to filter"},
		{Name: listFlagType, Kind: commandFlagString, ValueName: "<type>", Help: fmt.Sprintf("One of: %s", strings.Join(secretcontract.Names(), "|"))},
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
	Config:    commandConfigProjectOnly,
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
	return parsed.listOptions
}

func buildListQuery(opts listOptions) (secretsync.ListQuery, error) {
	var re *regexp.Regexp
	var selectedType secretprovider.SecretType

	if opts.nameRegex != "" {
		compiled, err := regexp.Compile(opts.nameRegex)
		if err != nil {
			return secretsync.ListQuery{}, usageError(fmt.Errorf("invalid --name-regex: %w", err))
		}
		re = compiled
	}

	if opts.secretType != "" {
		parsedType, err := parseSecretType(opts.secretType)
		if err != nil {
			return secretsync.ListQuery{}, usageError(fmt.Errorf("invalid --type: %w", err))
		}
		selectedType = parsedType
	}

	return secretsync.ListQuery{
		NameContains: opts.nameContains,
		NameRegex:    re,
		Path:         opts.path,
		Type:         selectedType,
	}, nil
}

func renderListOutput(ctx commandContext, asJSON bool, filtered []secretsync.ListRecord) error {
	if asJSON {
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
}

func runListParsed(ctx commandContext, parsed *parsedCommand) int {
	opts := parseListOptions(parsed)
	runtime := newCommandRuntime(ctx, parsed)
	if err := rejectUnexpectedArgs(parsed, "list"); err != nil {
		return runtime.writeStderrError(err)
	}
	query, err := buildListQuery(opts)
	if err != nil {
		return runtime.writeStderrError(err)
	}

	return runtime.executeWithConfigPolicy(parsed.configPolicy, func(_ *config.Loaded, service secretsync.Service) error {
		filtered, err := service.List(query)
		if err != nil {
			return err
		}

		return renderListOutput(ctx, opts.json, filtered)
	})
}
