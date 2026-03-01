package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

var pushCommandDef = commandDef{
	Name:    "push",
	Summary: "Push local files as new secret versions",
	Flags: []commandFlagDef{
		{Name: "all", Kind: commandFlagBool, Help: "Push all mapping entries with mode push|both (mode defaults to both)"},
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

func runPush(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, pushCommandDef)
}

func runPushParsed(ctx commandContext, parsed *parsedCommand) int {
	return newCommandRuntime(ctx, parsed).executeMapping(mappingCommandSpec{
		mode: "push",
		all:  parsed.Bool("all"),
		preflight: func(targets []secretsync.MappingTarget) error {
			if len(targets) > 1 && !parsed.Bool("yes") {
				return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
			}
			return nil
		},
		execute: func(service secretsync.Service, targets []secretsync.MappingTarget) error {
			results, err := service.Push(targets, secretsync.PushOptions{
				Description:     parsed.String("description"),
				DisablePrevious: parsed.Bool("disable-previous"),
				CreateMissing:   parsed.Bool("create-missing"),
			})
			if err != nil {
				return err
			}
			for _, item := range results {
				if _, err := fmt.Fprintf(ctx.stdout, "pushed %s (rev=%d)\n", item.Name, item.Revision); err != nil {
					return outputError(err)
				}
			}
			return nil
		},
	})
}
