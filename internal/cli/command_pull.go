package cli

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
	Config:    commandConfigValidated,
	RunParsed: runPullParsed,
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
	return runPullBatch(ctx, parsed, parsed.configPolicy, opts)
}
