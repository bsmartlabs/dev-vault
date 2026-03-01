package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

const (
	pullFlagAll       = "all"
	pullFlagOverwrite = "overwrite"
)

var pullCommandDef = commandDef{
	Name:    "pull",
	Summary: "Pull mapped -dev secrets to local files",
	Flags: []commandFlagDef{
		{Name: pullFlagAll, Kind: commandFlagBool, Help: "Pull all mapping entries with mode pull"},
		{Name: pullFlagOverwrite, Kind: commandFlagBool, Help: "Overwrite existing files"},
	},
	Doc: commandDoc{
		Synopsis: "dev-vault [--config <path>] [--profile <name>] pull (--all | <secret-dev> ...) [options]",
		Description: []string{
			"Pulls one or more secrets to disk based on .scw.json mapping.",
			"Secrets must exist in mapping and names must end with '-dev'.",
			"Pull reads the latest enabled secret version (Scaleway revision selector: latest_enabled).",
			"Pull writes files atomically and chmods them to 0600 (on Unix).",
			"Never prints secret payloads.",
			"",
			"Formats:",
			"  - mapping.format=raw writes secret bytes as-is.",
			"  - mapping.format=dotenv expects a JSON object payload and renders deterministic .env output.",
		},
		Examples: []string{
			"dev-vault pull bweb-env-bsmart-dev --overwrite",
			"dev-vault pull --all --overwrite",
			"dev-vault pull --config .scw.json bweb-env-bsmart-dev --overwrite",
			"dev-vault pull bweb-env-bsmart-dev --config .scw.json --overwrite",
		},
	},
	Config:           commandConfigValidated,
	NeedsRuntimeDeps: true,
	DecodeParsed:     decodePullParsed,
	RunParsed:        runPullParsed,
}

type pullOptions struct {
	all       bool
	overwrite bool
}

func decodePullParsed(parsed *parsedCommand, values parsedFlagValues) {
	parsed.pullOptions = pullOptions{
		all:       boolFlagValue(values, pullFlagAll),
		overwrite: boolFlagValue(values, pullFlagOverwrite),
	}
}

func runPullParsed(ctx commandContext, parsed *parsedCommand) int {
	return runPullBatch(ctx, parsed, parsed.pullOptions)
}

func reportPullBatchResults(ctx commandContext, result secretsync.PullBatchResult) error {
	return reportBatchResults(
		ctx,
		result.Succeeded,
		result.Failed,
		result.Summary,
		"pull",
		func(item secretsync.PullResult) string {
			return fmt.Sprintf("pulled %s -> %s (rev=%d type=%s)", item.Name, item.File, item.Revision, item.Type)
		},
	)
}

func runPullBatch(ctx commandContext, parsed *parsedCommand, opts pullOptions) int {
	return runOperationBatch(
		ctx,
		parsed,
		mapping.ModePull,
		opts.all,
		nil,
		func(service secretsync.Service, targets []mapping.Target) secretsync.PullBatchResult {
			return service.PullBatch(targets, opts.overwrite)
		},
		reportPullBatchResults,
	)
}
