package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

// withLoadedService centralizes command lifecycle orchestration:
// config/API bootstrap, warning emission, and consistent error-to-exit mapping.
func withLoadedService(
	ctx commandContext,
	parsed *parsedCommand,
	run func(loaded *config.Loaded, service commandService) error,
) int {
	loaded, api, err := loadAndOpenAPI(parsed.configPath, parsed.profileOverride, ctx.deps)
	if err != nil {
		runErr := runtimeError(err)
		fmt.Fprintln(ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}

	printConfigWarnings(ctx.stderr, loaded.Warnings)
	service := newCommandService(loaded, api, ctx.deps)
	if err := run(loaded, service); err != nil {
		fmt.Fprintln(ctx.stderr, err.Error())
		return exitCodeForError(err)
	}
	return 0
}

func withMappingCommand(
	ctx commandContext,
	parsed *parsedCommand,
	mode string,
	all bool,
	run func(service commandService, targets []mappingTarget) error,
) int {
	return withLoadedService(ctx, parsed, func(loaded *config.Loaded, service commandService) error {
		targets, err := selectMappingCommandTargets(loaded.Cfg.Mapping, all, parsed.fs.Args(), mode)
		if err != nil {
			return err
		}
		return run(service, targets)
	})
}

type mappingCommandSpec struct {
	mode      string
	all       bool
	preflight func(targets []mappingTarget) error
	execute   func(service commandService, targets []mappingTarget) error
}

func runMappingCommand(ctx commandContext, parsed *parsedCommand, spec mappingCommandSpec) int {
	return withMappingCommand(ctx, parsed, spec.mode, spec.all, func(service commandService, targets []mappingTarget) error {
		if spec.preflight != nil {
			if err := spec.preflight(targets); err != nil {
				return err
			}
		}
		return spec.execute(service, targets)
	})
}
