package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

func reportBatchResults[T any](
	ctx commandContext,
	succeeded []T,
	failed []secretsync.BatchFailure,
	summary secretsync.BatchSummary,
	operation string,
	renderSuccess func(T) string,
) error {
	for _, item := range succeeded {
		if _, err := fmt.Fprintln(ctx.stdout, renderSuccess(item)); err != nil {
			return outputError(err)
		}
	}
	for _, failure := range failed {
		if _, err := fmt.Fprintf(ctx.stderr, "failed %s %s: %v\n", operation, failure.Name, failure.Err); err != nil {
			return outputError(err)
		}
	}
	return summary.ErrorOrNil()
}
