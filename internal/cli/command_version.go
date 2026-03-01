package cli

import "fmt"

func runVersion(ctx commandContext, _ []string) int {
	fmt.Fprintf(ctx.stdout, "dev-vault %s (commit=%s date=%s)\n", ctx.deps.Version, ctx.deps.Commit, ctx.deps.Date)
	return 0
}
