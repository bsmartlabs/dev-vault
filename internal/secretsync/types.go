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

type MappingFormat string

const (
	MappingFormatRaw    MappingFormat = "raw"
	MappingFormatDotenv MappingFormat = "dotenv"
)

type MappingEntry struct {
	File   string
	Format MappingFormat
	Path   string
	Type   string
}

func MappingEntryFromConfig(entry config.MappingEntry) MappingEntry {
	return MappingEntry{
		File:   entry.File,
		Format: MappingFormat(entry.Format),
		Path:   entry.Path,
		Type:   entry.Type,
	}
}

func mappingFromConfigEntries(entries map[string]config.MappingEntry) map[string]MappingEntry {
	mapped := make(map[string]MappingEntry, len(entries))
	for name, entry := range entries {
		mapped[name] = MappingEntryFromConfig(entry)
	}
	return mapped
}

type MappingTarget struct {
	Name  string
	Entry MappingEntry
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
	Mapping map[string]MappingEntry
}

type PathResolver func(rootDir string, rel string) (string, error)

type Dependencies struct {
	Now         func() time.Time
	Hostname    func() (string, error)
	ResolvePath PathResolver
}

type Service struct {
	cfg         Config
	api         secretprovider.SecretAPI
	now         func() time.Time
	hostname    func() (string, error)
	resolvePath PathResolver
}

func NewFromLoaded(loaded *config.Loaded, api secretprovider.SecretAPI, deps Dependencies) Service {
	return New(Config{
		Root:    loaded.Root,
		Mapping: mappingFromConfigEntries(loaded.Cfg.Mapping),
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
	resolvePath := deps.ResolvePath
	if resolvePath == nil {
		resolvePath = config.ResolveFile
	}
	return Service{
		cfg:         cfg,
		api:         api,
		now:         now,
		hostname:    hostname,
		resolvePath: resolvePath,
	}
}
