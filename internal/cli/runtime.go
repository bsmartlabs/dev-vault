package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type mappingCommandSpec struct {
	mode      commandMode
	all       bool
	preflight func(targets []secretsync.MappingTarget) error
	execute   func(service secretsync.Service, targets []secretsync.MappingTarget) error
}

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
		_, _ = fmt.Fprintln(r.ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}

	if err := printConfigWarnings(r.ctx.stderr, loaded.Warnings); err != nil {
		runErr := outputError(err)
		_, _ = fmt.Fprintln(r.ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}
	service := secretsync.NewFromLoaded(loaded, api, secretsync.Dependencies{
		Now:      r.ctx.deps.Now,
		Hostname: r.ctx.deps.Hostname,
	})
	if err := run(loaded, service); err != nil {
		_, _ = fmt.Fprintln(r.ctx.stderr, err.Error())
		return exitCodeForError(err)
	}
	return 0
}

func (r commandRuntime) executeMapping(spec mappingCommandSpec) int {
	return r.execute(func(loaded *config.Loaded, service secretsync.Service) error {
		targets, err := selectMappingTargetsForMode(loaded.Cfg.Mapping, spec.all, r.parsed.fs.Args(), spec.mode)
		if err != nil {
			return err
		}
		if spec.preflight != nil {
			if err := spec.preflight(targets); err != nil {
				return err
			}
		}
		return spec.execute(service, targets)
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
