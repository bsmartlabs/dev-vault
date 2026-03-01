package cli

import "github.com/bsmartlabs/dev-vault/internal/secretsync"

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
	Config:    commandConfigValidated,
	RunParsed: runPushParsed,
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
	return runPushBatch(ctx, parsed, parsed.configPolicy, opts)
}
