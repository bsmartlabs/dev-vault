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
	Type         string
}

type listRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
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
		return nil, nil, fmt.Errorf("open scaleway api: %w", err)
	}
	return loaded, api, nil
}
