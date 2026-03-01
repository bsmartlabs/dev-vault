package cli

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

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

	req := ListSecretsInput{
		Region:    runtime.loaded.Cfg.Region,
		ProjectID: runtime.loaded.Cfg.ProjectID,
	}
	if query.Path != "" {
		req.Path = query.Path
	}
	secretTypes := supportedSecretTypes()
	if query.Type != "" {
		st, err := parseSecretType(query.Type)
		if err != nil {
			return nil, nil, usageError(fmt.Errorf("invalid --type: %w", err))
		}
		secretTypes = []string{st}
	}

	respSecrets, err := listSecretsByTypes(runtime.api, req, secretTypes)
	if err != nil {
		return nil, nil, runtimeError(err)
	}

	filtered := make([]listRecord, 0, len(respSecrets))
	for _, s := range respSecrets {
		if s == nil {
			continue
		}
		if !strings.HasSuffix(s.Name, "-dev") {
			continue
		}
		if query.Path != "" && s.Path != query.Path {
			continue
		}
		if len(query.NameContains) > 0 {
			miss := false
			for _, c := range query.NameContains {
				if !strings.Contains(s.Name, c) {
					miss = true
					break
				}
			}
			if miss {
				continue
			}
		}
		if query.NameRegex != nil && !query.NameRegex.MatchString(s.Name) {
			continue
		}
		filtered = append(filtered, listRecord{
			ID:   s.ID,
			Name: s.Name,
			Path: s.Path,
			Type: s.Type,
		})
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered, runtime.loaded.Warnings, nil
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
