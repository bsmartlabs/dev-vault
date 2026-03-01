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
	parsed *parsedCommand,
	mode mapping.Mode,
	all bool,
	preflight func([]mapping.Target) error,
	operation func(secretsync.Service, []mapping.Target) T,
	report func(commandContext, T) error,
) int {
	return runBatchCommand(ctx, parsed, batchCommandSpec{
		mode:      mode,
		all:       all,
		preflight: preflight,
		execute: func(service secretsync.Service, targets []mapping.Target) error {
			result := operation(service, targets)
			return report(ctx, result)
		},
	})
}

func runBatchCommand(ctx commandContext, parsed *parsedCommand, spec batchCommandSpec) int {
	runtime := newCommandRuntime(ctx, parsed)
	loaded, err := runtime.loadWithPolicy(parsed.configPolicy)
	if err != nil {
		return runtime.writeStderrError(err)
	}

	targets, err := runtime.selectMappingTargets(loaded, spec.mode, spec.all, spec.preflight, parsed.fs.Args())
	if err != nil {
		return runtime.writeStderrError(err)
	}

	service, err := runtime.newService(loaded)
	if err != nil {
		return runtime.writeStderrError(runtimeError(err))
	}

	if err := spec.execute(service, targets); err != nil {
		return runtime.writeStderrError(err)
	}
	return 0
}
