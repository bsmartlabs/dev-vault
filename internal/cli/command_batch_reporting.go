package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

func reportPullBatchResults(ctx commandContext, result secretsync.PullBatchResult) error {
	for _, item := range result.Succeeded {
		if _, err := fmt.Fprintf(ctx.stdout, "pulled %s -> %s (rev=%d type=%s)\n", item.Name, item.File, item.Revision, item.Type); err != nil {
			return outputError(err)
		}
	}
	for _, failure := range result.Failed {
		if _, err := fmt.Fprintf(ctx.stderr, "failed pull %s: %v\n", failure.Name, failure.Err); err != nil {
			return outputError(err)
		}
	}
	return result.Summary.ErrorOrNil()
}

func reportPushBatchResults(ctx commandContext, result secretsync.PushBatchResult) error {
	for _, item := range result.Succeeded {
		if _, err := fmt.Fprintf(ctx.stdout, "pushed %s (rev=%d)\n", item.Name, item.Revision); err != nil {
			return outputError(err)
		}
	}
	for _, failure := range result.Failed {
		if _, err := fmt.Fprintf(ctx.stderr, "failed push %s: %v\n", failure.Name, failure.Err); err != nil {
			return outputError(err)
		}
	}
	return result.Summary.ErrorOrNil()
}

func runPullBatch(ctx commandContext, parsed *parsedCommand, policy commandConfigPolicy, opts pullOptions) int {
	return newCommandRuntime(ctx, parsed).runMappingCommand(
		policy,
		mapping.ModePull,
		opts.all,
		nil,
		func(service secretsync.Service, targets []mapping.Target) error {
			result := service.PullBatch(targets, opts.overwrite)
			return reportPullBatchResults(ctx, result)
		},
	)
}

func runPushBatch(ctx commandContext, parsed *parsedCommand, policy commandConfigPolicy, opts pushOptions) int {
	preflight := func(targets []mapping.Target) error {
		if len(targets) > 1 && !opts.yes {
			return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
		}
		return nil
	}

	return newCommandRuntime(ctx, parsed).runMappingCommand(
		policy,
		mapping.ModePush,
		opts.all,
		preflight,
		func(service secretsync.Service, targets []mapping.Target) error {
			result := service.PushBatch(targets, opts.pushOptions())
			return reportPushBatchResults(ctx, result)
		},
	)
}
