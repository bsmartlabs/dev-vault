package cli

import (
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type commandService struct {
	cfg      commandServiceConfig
	api      secretprovider.SecretAPI
	now      func() time.Time
	hostname func() (string, error)
}

type commandServiceConfig struct {
	Root      string
	Region    string
	ProjectID string
	Mapping   map[string]config.MappingEntry
}

type pullResult struct {
	Name     string
	File     string
	Revision uint32
	Type     string
}

type pushOptions struct {
	Description     string
	DisablePrevious bool
	CreateMissing   bool
}

type pushResult struct {
	Name     string
	Revision uint32
}

func newCommandService(loaded *config.Loaded, api secretprovider.SecretAPI, deps Dependencies) commandService {
	return newCommandServiceWithConfig(commandServiceConfig{
		Root:      loaded.Root,
		Region:    loaded.Cfg.Region,
		ProjectID: loaded.Cfg.ProjectID,
		Mapping:   loaded.Cfg.Mapping,
	}, api, deps)
}

func newCommandServiceWithConfig(cfg commandServiceConfig, api secretprovider.SecretAPI, deps Dependencies) commandService {
	return commandService{
		cfg:      cfg,
		api:      api,
		now:      deps.Now,
		hostname: deps.Hostname,
	}
}

func (s commandService) projectScope() secretProjectScope {
	return secretProjectScope{
		Region:    s.cfg.Region,
		ProjectID: s.cfg.ProjectID,
	}
}
