package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/pathpolicy"
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
	return r.runWithLoaded(func(loaded *config.Loaded) error {
		service, err := r.newService(loaded)
		if err != nil {
			return runtimeError(err)
		}
		return run(loaded, service)
	})
}

func (r commandRuntime) runWithLoaded(run func(loaded *config.Loaded) error) int {
	loaded, err := loadConfig(r.parsed.configPath, r.ctx.deps)
	if err != nil {
		return r.writeStderrError(runtimeError(err))
	}
	if err := run(loaded); err != nil {
		return r.writeStderrError(err)
	}
	return 0
}

func (r commandRuntime) newService(loaded *config.Loaded) (secretsync.Service, error) {
	api, err := openAPIForLoaded(loaded, r.parsed.profileOverride, r.ctx.deps)
	if err != nil {
		return secretsync.Service{}, err
	}
	service, err := secretsync.New(secretsync.Config{
		Root: loaded.Root,
	}, api, secretsync.Dependencies{
		Now:         r.ctx.deps.Now,
		Hostname:    r.ctx.deps.Hostname,
		ResolvePath: pathpolicy.ResolveProjectFile,
	})
	if err != nil {
		return secretsync.Service{}, fmt.Errorf("init secret sync service: %w", err)
	}
	return service, nil
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
	return r.runWithLoaded(func(loaded *config.Loaded) error {
		targets, err := selectMappingTargetsForMode(loaded.Cfg.Mapping, all, r.parsed.fs.Args(), mode)
		if err != nil {
			return err
		}
		if preflight != nil {
			if err := preflight(targets); err != nil {
				return err
			}
		}
		service, err := r.newService(loaded)
		if err != nil {
			return runtimeError(err)
		}
		return execute(service, targets)
	})
}

func loadConfig(configPath string, deps Dependencies) (*config.Loaded, error) {
	wd, err := deps.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	loaded, err := config.Load(wd, configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return loaded, nil
}

func openAPIForLoaded(loaded *config.Loaded, profileOverride string, deps Dependencies) (secretprovider.SecretAPI, error) {
	api, err := deps.OpenSecretAPI(loaded.Cfg, profileOverride)
	if err != nil {
		return nil, fmt.Errorf("open secret api: %w", err)
	}
	return api, nil
}
