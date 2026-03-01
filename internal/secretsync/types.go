package secretsync

import (
	"os"
	"regexp"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type ListQuery struct {
	NameContains []string
	NameRegex    *regexp.Regexp
	Path         string
	Type         secretprovider.SecretType
}

type ListRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type MappingTarget struct {
	Name  string
	Entry config.MappingEntry
}

type PullResult struct {
	Name     string
	File     string
	Revision uint32
	Type     string
}

type PushOptions struct {
	Description     string
	DisablePrevious bool
	CreateMissing   bool
}

type PushResult struct {
	Name     string
	Revision uint32
}

type Config struct {
	Root    string
	Mapping map[string]config.MappingEntry
}

type Dependencies struct {
	Now      func() time.Time
	Hostname func() (string, error)
}

type Service struct {
	cfg      Config
	api      secretprovider.SecretAPI
	now      func() time.Time
	hostname func() (string, error)
}

func NewFromLoaded(loaded *config.Loaded, api secretprovider.SecretAPI, deps Dependencies) Service {
	return New(Config{
		Root:    loaded.Root,
		Mapping: loaded.Cfg.Mapping,
	}, api, deps)
}

func New(cfg Config, api secretprovider.SecretAPI, deps Dependencies) Service {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	hostname := deps.Hostname
	if hostname == nil {
		hostname = os.Hostname
	}
	return Service{
		cfg:      cfg,
		api:      api,
		now:      now,
		hostname: hostname,
	}
}
