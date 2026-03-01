package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

var pullCommandDef = commandDef{
	Name:    "pull",
	Summary: "Pull mapped -dev secrets to local files",
	Flags: []commandFlagDef{
		{Name: "all", Kind: commandFlagBool, Help: "Pull all mapping entries with mode pull"},
		{Name: "overwrite", Kind: commandFlagBool, Help: "Overwrite existing files"},
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
	RunParsed: runPullParsed,
}

var pullBatchOperation = mappingBatchOperation[secretsync.PullResult]{
	mode: commandModePull,
	run: func(service secretsync.Service, parsed *parsedCommand, targets []secretsync.MappingTarget) (batchRunResult[secretsync.PullResult], error) {
		opts := parsePullOptions(parsed)
		result := service.PullBatch(targets, opts.overwrite)
		return batchRunResult[secretsync.PullResult]{
			successes: result.Succeeded,
			failures:  result.Failed,
			summary:   result.Summary,
		}, nil
	},
	callbacks: batchReportCallbacks[secretsync.PullResult]{
		SuccessLine: func(item secretsync.PullResult) string {
			return fmt.Sprintf("pulled %s -> %s (rev=%d type=%s)", item.Name, item.File, item.Revision, item.Type)
		},
		FailureLine: func(failure secretsync.BatchFailure) string {
			return fmt.Sprintf("failed pull %s: %v", failure.Name, failure.Err)
		},
	},
}

type pullOptions struct {
	all       bool
	overwrite bool
}

func parsePullOptions(parsed *parsedCommand) pullOptions {
	return pullOptions{
		all:       parsed.Bool("all"),
		overwrite: parsed.Bool("overwrite"),
	}
}

func runPullParsed(ctx commandContext, parsed *parsedCommand) int {
	opts := parsePullOptions(parsed)
	return runMappingBatchOperation(ctx, parsed, opts.all, pullBatchOperation)
}
