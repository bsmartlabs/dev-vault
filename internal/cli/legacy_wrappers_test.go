package cli

import "io"

func runList(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, listCommandDef)
}

func runPull(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, pullCommandDef)
}

func runPush(ctx commandContext, argv []string) int {
	return runCommand(ctx, argv, pushCommandDef)
}

func printVersionUsage(w io.Writer) error {
	return printCommandUsage(w, versionCommandDef)
}

func printListUsage(w io.Writer) error {
	return printCommandUsage(w, listCommandDef)
}

func printPullUsage(w io.Writer) error {
	return printCommandUsage(w, pullCommandDef)
}

func printPushUsage(w io.Writer) error {
	return printCommandUsage(w, pushCommandDef)
}
