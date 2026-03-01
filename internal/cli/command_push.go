package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

var pushCommandDef = commandDef{
	Name:    "push",
	Summary: "Push local files as new secret versions",
	Flags: []commandFlagDef{
		{Name: "all", Kind: commandFlagBool, Help: "Push all mapping entries with mode push"},
		{Name: "yes", Kind: commandFlagBool, Help: "Confirm batch push (required when pushing more than one secret)"},
		{Name: "disable-previous", Kind: commandFlagBool, Help: "Disable previous enabled version when creating a new version"},
		{Name: "description", Kind: commandFlagString, ValueName: "<text>", Help: "Description for the new version (optional)"},
		{Name: "create-missing", Kind: commandFlagBool, Help: "Create missing secrets (requires mapping.type)"},
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
	RunParsed: runPushParsed,
}

var pushBatchOperation = mappingBatchOperation[secretsync.PushResult]{
	mode: commandModePush,
	preflight: func(parsed *parsedCommand, targets []secretsync.MappingTarget) error {
		opts := parsePushOptions(parsed)
		if len(targets) > 1 && !opts.yes {
			return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
		}
		return nil
	},
	run: func(service secretsync.Service, parsed *parsedCommand, targets []secretsync.MappingTarget) (batchRunResult[secretsync.PushResult], error) {
		opts := parsePushOptions(parsed)
		result := service.PushBatch(targets, opts.pushOptions())
		return batchRunResult[secretsync.PushResult]{
			successes: result.Succeeded,
			failures:  result.Failed,
			summary:   result.Summary,
		}, nil
	},
	callbacks: batchReportCallbacks[secretsync.PushResult]{
		SuccessLine: func(item secretsync.PushResult) string {
			return fmt.Sprintf("pushed %s (rev=%d)", item.Name, item.Revision)
		},
		FailureLine: func(failure secretsync.BatchFailure) string {
			return fmt.Sprintf("failed push %s: %v", failure.Name, failure.Err)
		},
	},
}

type pushOptions struct {
	all             bool
	yes             bool
	disablePrevious bool
	description     string
	createMissing   bool
}

func (o pushOptions) pushOptions() secretsync.PushOptions {
	return secretsync.PushOptions{
		Description:     o.description,
		DisablePrevious: o.disablePrevious,
		CreateMissing:   o.createMissing,
	}
}

func parsePushOptions(parsed *parsedCommand) pushOptions {
	return pushOptions{
		all:             parsed.Bool("all"),
		yes:             parsed.Bool("yes"),
		disablePrevious: parsed.Bool("disable-previous"),
		description:     parsed.String("description"),
		createMissing:   parsed.Bool("create-missing"),
	}
}

func runPushParsed(ctx commandContext, parsed *parsedCommand) int {
	opts := parsePushOptions(parsed)
	return runMappingBatchOperation(ctx, parsed, opts.all, pushBatchOperation)
}
