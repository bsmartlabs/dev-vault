package cli

import "fmt"

var versionCommandDef = commandDef{
	Name:             "version",
	Summary:          "Print build version information",
	Config:           0,
	NeedsRuntimeDeps: false,
	Doc: commandDoc{
		Synopsis: "dev-vault version",
		Description: []string{
			"Prints the build version/commit/date.",
		},
	},
	RunParsed: runVersionParsed,
}

func runVersionParsed(ctx commandContext, parsed *parsedCommand) int {
	if err := rejectUnexpectedArgs(parsed, "version"); err != nil {
		return newCommandRuntime(ctx, parsed).writeStderrError(err)
	}
	if _, err := fmt.Fprintf(ctx.stdout, "dev-vault %s (commit=%s date=%s)\n", ctx.deps.Version, ctx.deps.Commit, ctx.deps.Date); err != nil {
		return exitCodeForError(outputError(err))
	}
	return 0
}
