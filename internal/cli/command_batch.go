package cli

import (
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type batchCommandSpec struct {
	mode      mapping.Mode
	all       bool
	preflight func([]mapping.Target) error
	execute   func(secretsync.Service, []mapping.Target) error
}

func runOperationBatch[T any](
	ctx commandContext,
	params commandParams,
	mode mapping.Mode,
	all bool,
	preflight func([]mapping.Target) error,
	operation func(secretsync.Service, []mapping.Target) T,
	report func(commandContext, T) error,
) error {
	return runBatchCommand(ctx, params, batchCommandSpec{
		mode:      mode,
		all:       all,
		preflight: preflight,
		execute: func(service secretsync.Service, targets []mapping.Target) error {
			result := operation(service, targets)
			return report(ctx, result)
		},
	})
}

func runBatchCommand(ctx commandContext, params commandParams, spec batchCommandSpec) error {
	runtime := newCommandRuntime(ctx, params)
	loaded, err := runtime.loadWithPolicy(params.configPolicy)
	if err != nil {
		return err
	}

	targets, err := runtime.selectMappingTargets(loaded, spec.mode, spec.all, spec.preflight)
	if err != nil {
		return err
	}

	service, err := runtime.newService(loaded)
	if err != nil {
		return runtimeError(err)
	}

	if err := spec.execute(service, targets); err != nil {
		return runtimeError(err)
	}
	return nil
}
