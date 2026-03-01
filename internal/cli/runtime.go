package cli

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/cli/selection"
	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

var (
	errRuntimeOpenSecretAPI       = errors.New("open secret api")
	errRuntimeInitSecretSyncError = errors.New("init secret sync service")
)

type commandRuntime struct {
	ctx    commandContext
	parsed *parsedCommand
}

type configLoader func(configPath string, deps Dependencies) (*config.Loaded, error)
type projectConfigLoader func(startDir, explicitPath string) (*config.Loaded, error)

type runtimeResources struct {
	loaded  *config.Loaded
	service secretsync.Service
}

func newCommandRuntime(ctx commandContext, parsed *parsedCommand) commandRuntime {
	return commandRuntime{ctx: ctx, parsed: parsed}
}

func (r commandRuntime) loadWithPolicy(policy commandConfigPolicy) (*config.Loaded, error) {
	loader, err := configLoaderForPolicy(policy)
	if err != nil {
		return nil, runtimeError(err)
	}
	loaded, err := loader(r.parsed.configPath, r.ctx.deps)
	if err != nil {
		return nil, runtimeError(err)
	}
	return loaded, nil
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
		ResolvePath: r.ctx.deps.ResolveProjectPath,
	})
	if err != nil {
		return secretsync.Service{}, fmt.Errorf("%w: %w", errRuntimeInitSecretSyncError, err)
	}
	return service, nil
}

func (r commandRuntime) prepareResources(policy commandConfigPolicy) (*runtimeResources, error) {
	loaded, err := r.loadWithPolicy(policy)
	if err != nil {
		return nil, err
	}
	service, err := r.newService(loaded)
	if err != nil {
		return nil, runtimeError(err)
	}
	return &runtimeResources{
		loaded:  loaded,
		service: service,
	}, nil
}

func (r commandRuntime) writeStderrError(err error) int {
	if _, writeErr := fmt.Fprintln(r.ctx.stderr, err.Error()); writeErr != nil {
		return exitCodeForError(outputError(writeErr))
	}
	return exitCodeForError(err)
}

func (r commandRuntime) selectMappingTargets(
	loaded *config.Loaded,
	mode mapping.Mode,
	all bool,
	preflight func(targets []mapping.Target) error,
	argv []string,
) ([]mapping.Target, error) {
	targets, err := selection.SelectTargetsForMode(loaded.Cfg.Mapping, all, argv, mode)
	if err != nil {
		return nil, usageError(err)
	}
	if preflight != nil {
		if err := preflight(targets); err != nil {
			return nil, err
		}
	}
	return targets, nil
}

func configLoaderForPolicy(policy commandConfigPolicy) (configLoader, error) {
	switch policy {
	case commandConfigProjectOnly:
		return loadProjectConfig, nil
	case commandConfigValidated:
		return loadConfig, nil
	default:
		return nil, fmt.Errorf("internal error: unsupported command config policy %d", policy)
	}
}

func loadConfig(configPath string, deps Dependencies) (*config.Loaded, error) {
	return loadConfigWithLoader(configPath, deps, config.Load)
}

func loadProjectConfig(configPath string, deps Dependencies) (*config.Loaded, error) {
	return loadConfigWithLoader(configPath, deps, config.LoadProject)
}

func loadConfigWithLoader(configPath string, deps Dependencies, loader projectConfigLoader) (*config.Loaded, error) {
	loaded, err := loader(".", configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return loaded, nil
}

func openAPIForLoaded(loaded *config.Loaded, profileOverride string, deps Dependencies) (secretprovider.SecretAPI, error) {
	api, err := deps.OpenSecretAPI(loaded.Cfg, profileOverride)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRuntimeOpenSecretAPI, err)
	}
	return api, nil
}
