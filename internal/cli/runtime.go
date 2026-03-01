package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type commandRuntime struct {
	ctx    commandContext
	parsed *parsedCommand
}

func newCommandRuntime(ctx commandContext, parsed *parsedCommand) commandRuntime {
	return commandRuntime{ctx: ctx, parsed: parsed}
}

func (r commandRuntime) execute(run func(loaded *config.Loaded, service secretsync.Service) error) int {
	loaded, api, err := loadAndOpenAPI(r.parsed.configPath, r.parsed.profileOverride, r.ctx.deps)
	if err != nil {
		runErr := runtimeError(err)
		return r.writeStderrError(runErr)
	}

	service, err := secretsync.New(secretsync.Config{
		Root: loaded.Root,
	}, api, secretsync.Dependencies{
		Now:         r.ctx.deps.Now,
		Hostname:    r.ctx.deps.Hostname,
		ResolvePath: config.ResolveFile,
	})
	if err != nil {
		return r.writeStderrError(runtimeError(fmt.Errorf("init secret sync service: %w", err)))
	}
	if err := run(loaded, service); err != nil {
		return r.writeStderrError(err)
	}
	return 0
}

func (r commandRuntime) writeStderrError(err error) int {
	if _, writeErr := fmt.Fprintln(r.ctx.stderr, err.Error()); writeErr != nil {
		return exitCodeForError(outputError(writeErr))
	}
	return exitCodeForError(err)
}

func (r commandRuntime) runMappingCommand(
	mode mapping.Mode,
	all bool,
	preflight func(targets []secretsync.MappingTarget) error,
	execute func(service secretsync.Service, targets []secretsync.MappingTarget) error,
) int {
	return r.execute(func(loaded *config.Loaded, service secretsync.Service) error {
		targets, err := selectMappingTargetsForMode(loaded.Cfg.Mapping, all, r.parsed.fs.Args(), mode)
		if err != nil {
			return err
		}
		if preflight != nil {
			if err := preflight(targets); err != nil {
				return err
			}
		}
		return execute(service, targets)
	})
}

func loadAndOpenAPI(configPath, profileOverride string, deps Dependencies) (*config.Loaded, secretprovider.SecretAPI, error) {
	wd, err := deps.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("getwd: %w", err)
	}
	loaded, err := config.Load(wd, configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	api, err := deps.OpenSecretAPI(loaded.Cfg, profileOverride)
	if err != nil {
		return nil, nil, fmt.Errorf("open secret api: %w", err)
	}
	return loaded, api, nil
}
