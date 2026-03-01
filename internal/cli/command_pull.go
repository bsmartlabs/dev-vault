package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

var pullCommandDef = commandDef{
	Name:    "pull",
	Summary: "Pull mapped -dev secrets to local files",
	Flags: []commandFlagDef{
		{Name: "all", Kind: commandFlagBool, Help: "Pull all mapping entries with mode pull|both (mode defaults to both)"},
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

func runPull(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, pullCommandDef)
}

func runPullParsed(ctx commandContext, parsed *parsedCommand) int {
	return newCommandRuntime(ctx, parsed).executeMapping(mappingCommandSpec{
		mode: "pull",
		all:  parsed.Bool("all"),
		execute: func(service secretsync.Service, targets []secretsync.MappingTarget) error {
			results, err := service.Pull(targets, parsed.Bool("overwrite"))
			if err != nil {
				return err
			}
			for _, item := range results {
				if _, err := fmt.Fprintf(ctx.stdout, "pulled %s -> %s (rev=%d type=%s)\n", item.Name, item.File, item.Revision, item.Type); err != nil {
					return outputError(err)
				}
			}
			return nil
		},
	})
}
