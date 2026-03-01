package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type batchReportCallbacks[T any] struct {
	SuccessLine func(T) string
	FailureLine func(secretsync.BatchFailure) string
}

type batchRunResult[T any] struct {
	successes []T
	failures  []secretsync.BatchFailure
	summary   secretsync.BatchSummary
}

type mappingBatchOperation[T any, O any] struct {
	mode      mapping.Mode
	preflight func(opts O, targets []secretsync.MappingTarget) error
	run       func(service secretsync.Service, targets []secretsync.MappingTarget, opts O) (batchRunResult[T], error)
	callbacks batchReportCallbacks[T]
}

func reportBatchResults[T any](result batchRunResult[T], ctx commandContext, callbacks batchReportCallbacks[T]) error {
	for _, item := range result.successes {
		if _, err := fmt.Fprintln(ctx.stdout, callbacks.SuccessLine(item)); err != nil {
			return outputError(err)
		}
	}

	if result.summary.ErrorOrNil() == nil {
		return nil
	}
	for _, failure := range result.failures {
		if _, err := fmt.Fprintln(ctx.stderr, callbacks.FailureLine(failure)); err != nil {
			return outputError(err)
		}
	}
	return result.summary.ErrorOrNil()
}

func runMappingBatchOperation[T any, O any](ctx commandContext, parsed *parsedCommand, all bool, opts O, op mappingBatchOperation[T, O]) int {
	var preflight func(targets []secretsync.MappingTarget) error
	if op.preflight != nil {
		preflight = func(targets []secretsync.MappingTarget) error {
			return op.preflight(opts, targets)
		}
	}
	return newCommandRuntime(ctx, parsed).runMappingCommand(
		op.mode,
		all,
		preflight,
		func(service secretsync.Service, targets []secretsync.MappingTarget) error {
			result, err := op.run(service, targets, opts)
			if err != nil {
				return err
			}
			return reportBatchResults(result, ctx, op.callbacks)
		},
	)
}
