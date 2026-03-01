package cli

import "fmt"

var versionCommandDef = commandDef{
	Name:    "version",
	Summary: "Print build version information",
	Doc: commandDoc{
		Synopsis: "dev-vault version",
		Description: []string{
			"Prints the build version/commit/date.",
		},
	},
	RunParsed: runVersionParsed,
}

func runVersionParsed(ctx commandContext, _ *parsedCommand) int {
	if _, err := fmt.Fprintf(ctx.stdout, "dev-vault %s (commit=%s date=%s)\n", ctx.deps.Version, ctx.deps.Commit, ctx.deps.Date); err != nil {
		return exitCodeForError(outputError(err))
	}
	return 0
}
