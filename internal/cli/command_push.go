package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

const (
	pushFlagAll             = "all"
	pushFlagYes             = "yes"
	pushFlagDisablePrevious = "disable-previous"
	pushFlagDescription     = "description"
	pushFlagCreateMissing   = "create-missing"
)

var pushCommandDef = commandDef{
	Name:    "push",
	Summary: "Push local files as new secret versions",
	Flags: []commandFlagDef{
		{Name: pushFlagAll, Kind: commandFlagBool, Help: "Push all mapping entries with mode push"},
		{Name: pushFlagYes, Kind: commandFlagBool, Help: "Confirm batch push (required when pushing more than one secret)"},
		{Name: pushFlagDisablePrevious, Kind: commandFlagBool, Help: "Disable previous enabled version when creating a new version"},
		{Name: pushFlagDescription, Kind: commandFlagString, ValueName: "<text>", Help: "Description for the new version (optional)"},
		{Name: pushFlagCreateMissing, Kind: commandFlagBool, Help: "Create missing secrets (requires mapping.type)"},
	},
	Doc: commandDoc{
		Synopsis: "dev-vault [--config <path>] [--profile <name>] push (--all | <secret-dev> ...) [options]",
		Description: []string{
			"Pushes one or more secrets from disk to Scaleway Secret Manager as a new version.",
			"Secrets must exist in mapping and names must end with '-dev'.",
			"Never prints secret payloads.",
			"",
			"Formats:",
			"  - mapping.format=raw reads file bytes as-is.",
			"  - mapping.format=dotenv reads a .env file and uploads a JSON payload.",
		},
		Notes: []string{
			"--create-missing creates the secret if absent (requires mapping.type).",
			"Secret creation uses mapping.path (default '/').",
			"If more than one secret is being pushed, you must pass --yes.",
		},
		Examples: []string{
			"dev-vault push bweb-env-bsmart-dev",
			"dev-vault push bweb-env-bsmart-dev --description 'local refresh'",
			"dev-vault push --all --yes",
			"dev-vault push --config .scw.json --all --yes --disable-previous",
		},
	},
	Config:           commandConfigValidated,
	NeedsRuntimeDeps: true,
	DecodeParsed:     decodePushParsed,
	RunParsed:        runPushParsed,
}

type pushOptions struct {
	all             bool
	yes             bool
	disablePrevious bool
	description     string
	createMissing   bool
}

func decodePushParsed(parsed *parsedCommand, values parsedFlagValues) {
	parsed.pushOptions = pushOptions{
		all:             boolFlagValue(values, pushFlagAll),
		yes:             boolFlagValue(values, pushFlagYes),
		disablePrevious: boolFlagValue(values, pushFlagDisablePrevious),
		description:     stringFlagValue(values, pushFlagDescription),
		createMissing:   boolFlagValue(values, pushFlagCreateMissing),
	}
}

func (o pushOptions) toServicePushOptions() secretsync.PushOptions {
	return secretsync.PushOptions{
		Description:     o.description,
		DisablePrevious: o.disablePrevious,
		CreateMissing:   o.createMissing,
	}
}

func runPushParsed(ctx commandContext, parsed *parsedCommand) int {
	return runPushBatch(ctx, parsed, parsed.pushOptions)
}

func reportPushBatchResults(ctx commandContext, result secretsync.PushBatchResult) error {
	return reportBatchResults(
		ctx,
		result.Succeeded,
		result.Failed,
		result.Summary,
		"push",
		func(item secretsync.PushResult) string {
			return fmt.Sprintf("pushed %s (rev=%d)", item.Name, item.Revision)
		},
	)
}

func runPushBatch(ctx commandContext, parsed *parsedCommand, opts pushOptions) int {
	return runOperationBatch(
		ctx,
		parsed,
		mapping.ModePush,
		opts.all,
		func(targets []mapping.Target) error {
			if len(targets) > 1 && !opts.yes {
				return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
			}
			return nil
		},
		func(service secretsync.Service, targets []mapping.Target) secretsync.PushBatchResult {
			return service.PushBatch(targets, opts.toServicePushOptions())
		},
		reportPushBatchResults,
	)
}
