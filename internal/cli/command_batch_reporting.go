package cli

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type batchReportCallbacks[T any] struct {
	SuccessLine func(T) string
	FailureLine func(secretsync.BatchFailure) string
}

type mappingBatchOperation[T any, O any] struct {
	mode      mapping.Mode
	preflight func(opts O, targets []secretsync.MappingTarget) error
	run       func(service secretsync.Service, targets []secretsync.MappingTarget, opts O) (secretsync.BatchResult[T], error)
	callbacks batchReportCallbacks[T]
}

func (op mappingBatchOperation[T, O]) validate() error {
	if op.run == nil {
		return errors.New("batch operation run callback is required")
	}
	if op.callbacks.SuccessLine == nil {
		return errors.New("batch operation success callback is required")
	}
	if op.callbacks.FailureLine == nil {
		return errors.New("batch operation failure callback is required")
	}
	return nil
}

func reportBatchResults[T any](result secretsync.BatchResult[T], runErr error, ctx commandContext, callbacks batchReportCallbacks[T]) error {
	for _, item := range result.Succeeded {
		if _, err := fmt.Fprintln(ctx.stdout, callbacks.SuccessLine(item)); err != nil {
			return outputError(err)
		}
	}
	for _, failure := range result.Failed {
		if _, err := fmt.Fprintln(ctx.stderr, callbacks.FailureLine(failure)); err != nil {
			return outputError(err)
		}
	}
	if runErr != nil {
		return runErr
	}
	return nil
}

func runMappingBatchOperation[T any, O any](ctx commandContext, parsed *parsedCommand, all bool, opts O, op mappingBatchOperation[T, O]) int {
	if err := op.validate(); err != nil {
		return newCommandRuntime(ctx, parsed).writeStderrError(runtimeError(err))
	}

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
			return reportBatchResults(result, err, ctx, op.callbacks)
		},
	)
}
