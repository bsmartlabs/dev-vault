package cli

import (
	"fmt"
	"regexp"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

type commandRuntime struct {
	loaded  *config.Loaded
	api     SecretAPI
	service commandService
}

type listQuery struct {
	NameContains []string
	NameRegex    *regexp.Regexp
	Path         string
	Type         string
}

type listRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

func buildCommandRuntime(configPath, profileOverride string, deps Dependencies) (*commandRuntime, error) {
	loaded, api, err := loadAndOpenAPI(configPath, profileOverride, deps)
	if err != nil {
		return nil, err
	}
	return &commandRuntime{
		loaded:  loaded,
		api:     api,
		service: newCommandService(loaded, api, deps),
	}, nil
}

func executeList(configPath, profileOverride string, deps Dependencies, query listQuery) ([]listRecord, []string, error) {
	runtime, err := buildCommandRuntime(configPath, profileOverride, deps)
	if err != nil {
		return nil, nil, runtimeError(err)
	}

	filtered, err := runtime.service.list(query)
	return filtered, runtime.loaded.Warnings, err
}

func executePull(configPath, profileOverride string, deps Dependencies, all bool, positional []string, overwrite bool) ([]pullResult, []string, error) {
	runtime, err := buildCommandRuntime(configPath, profileOverride, deps)
	if err != nil {
		return nil, nil, runtimeError(err)
	}
	targets, err := selectMappingTargets(runtime.loaded.Cfg.Mapping, all, positional, "pull")
	if err != nil {
		return nil, runtime.loaded.Warnings, err
	}
	results, err := runtime.service.pull(targets, overwrite)
	return results, runtime.loaded.Warnings, err
}

func executePush(configPath, profileOverride string, deps Dependencies, all bool, positional []string, yes bool, options pushOptions) ([]pushResult, []string, error) {
	runtime, err := buildCommandRuntime(configPath, profileOverride, deps)
	if err != nil {
		return nil, nil, runtimeError(err)
	}
	targets, err := selectMappingTargets(runtime.loaded.Cfg.Mapping, all, positional, "push")
	if err != nil {
		return nil, runtime.loaded.Warnings, err
	}
	if len(targets) > 1 && !yes {
		return nil, runtime.loaded.Warnings, usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
	}
	results, err := runtime.service.push(targets, options)
	return results, runtime.loaded.Warnings, err
}

func loadAndOpenAPI(configPath, profileOverride string, deps Dependencies) (*config.Loaded, SecretAPI, error) {
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
		return nil, nil, fmt.Errorf("open scaleway api: %w", err)
	}
	return loaded, api, nil
}
