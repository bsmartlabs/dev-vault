package cli

import (
	"fmt"
	"path/filepath"

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

type configLoader func(configPath string, deps Dependencies) (*config.Loaded, error)
type projectConfigLoader func(startDir, explicitPath string) (*config.Loaded, error)

func newCommandRuntime(ctx commandContext, parsed *parsedCommand) commandRuntime {
	return commandRuntime{ctx: ctx, parsed: parsed}
}

func (r commandRuntime) executeWithConfigLoader(loader configLoader, run func(loaded *config.Loaded, service secretsync.Service) error) int {
	return r.runWithLoaded(loader, func(loaded *config.Loaded) error {
		service, err := r.newService(loaded)
		if err != nil {
			return runtimeError(err)
		}
		return run(loaded, service)
	})
}

func (r commandRuntime) executeWithConfigPolicy(policy commandConfigPolicy, run func(loaded *config.Loaded, service secretsync.Service) error) int {
	loader, err := configLoaderForPolicy(policy)
	if err != nil {
		return r.writeStderrError(runtimeError(err))
	}
	return r.executeWithConfigLoader(loader, run)
}

func (r commandRuntime) runWithLoaded(loader configLoader, run func(loaded *config.Loaded) error) int {
	loaded, err := loader(r.parsed.configPath, r.ctx.deps)
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
	policy commandConfigPolicy,
	mode mapping.Mode,
	all bool,
	preflight func(targets []mapping.Target) error,
	execute func(service secretsync.Service, targets []mapping.Target) error,
) int {
	loader, err := configLoaderForPolicy(policy)
	if err != nil {
		return r.writeStderrError(runtimeError(err))
	}
	return r.runWithLoaded(loader, func(loaded *config.Loaded) error {
		targets, err := mapping.SelectTargetsForMode(loaded.Cfg.Mapping, all, r.parsed.fs.Args(), mode)
		if err != nil {
			return usageError(err)
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

func configLoaderForPolicy(policy commandConfigPolicy) (configLoader, error) {
	switch policy {
	case commandConfigValidated:
		return loadConfig, nil
	case commandConfigProjectOnly:
		return loadProjectConfig, nil
	default:
		return nil, fmt.Errorf("unsupported command config policy: %d", policy)
	}
}

func loadConfig(configPath string, deps Dependencies) (*config.Loaded, error) {
	return loadConfigWithLoader(configPath, deps, config.Load)
}

func loadProjectConfig(configPath string, deps Dependencies) (*config.Loaded, error) {
	return loadConfigWithLoader(configPath, deps, config.LoadProject)
}

func loadConfigWithLoader(configPath string, deps Dependencies, loader projectConfigLoader) (*config.Loaded, error) {
	if configPath != "" && filepath.IsAbs(configPath) {
		loaded, err := loader(string(filepath.Separator), configPath)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		return loaded, nil
	}
	wd, err := deps.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	loaded, err := loader(wd, configPath)
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
