package cli

import (
	"fmt"
	"regexp"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type listQuery struct {
	NameContains []string
	NameRegex    *regexp.Regexp
	Path         string
	Type         secretprovider.SecretType
}

type listRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type mappingCommandSpec struct {
	mode      string
	all       bool
	preflight func(targets []mappingTarget) error
	execute   func(service commandService, targets []mappingTarget) error
}

type commandRuntime struct {
	ctx    commandContext
	parsed *parsedCommand
}

func newCommandRuntime(ctx commandContext, parsed *parsedCommand) commandRuntime {
	return commandRuntime{ctx: ctx, parsed: parsed}
}

func (r commandRuntime) execute(run func(loaded *config.Loaded, service commandService) error) int {
	loaded, api, err := loadAndOpenAPI(r.parsed.configPath, r.parsed.profileOverride, r.ctx.deps)
	if err != nil {
		runErr := runtimeError(err)
		fmt.Fprintln(r.ctx.stderr, runErr.Error())
		return exitCodeForError(runErr)
	}

	printConfigWarnings(r.ctx.stderr, loaded.Warnings)
	service := newCommandService(loaded, api, r.ctx.deps)
	if err := run(loaded, service); err != nil {
		fmt.Fprintln(r.ctx.stderr, err.Error())
		return exitCodeForError(err)
	}
	return 0
}

func (r commandRuntime) executeMapping(spec mappingCommandSpec) int {
	return r.execute(func(loaded *config.Loaded, service commandService) error {
		targets, err := selectMappingCommandTargets(loaded.Cfg.Mapping, spec.all, r.parsed.fs.Args(), spec.mode)
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
