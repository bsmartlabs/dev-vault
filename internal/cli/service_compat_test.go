package cli

import (
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type commandServiceConfig struct {
	Root    string
	Mapping map[string]config.MappingEntry
}

type listQuery = secretsync.ListQuery
type listRecord = secretsync.ListRecord
type secretLookupMissError = secretsync.SecretLookupMissError

type commandService struct {
	cfg      commandServiceConfig
	api      secretprovider.SecretAPI
	now      func() time.Time
	hostname func() (string, error)
	inner    secretsync.Service
}

func newCommandService(loaded *config.Loaded, api secretprovider.SecretAPI, deps Dependencies) commandService {
	return newCommandServiceWithConfig(commandServiceConfig{
		Root:    loaded.Root,
		Mapping: loaded.Cfg.Mapping,
	}, api, deps)
}

func newCommandServiceWithConfig(cfg commandServiceConfig, api secretprovider.SecretAPI, deps Dependencies) commandService {
	syncDeps := secretsync.Dependencies{
		Now:         deps.Now,
		Hostname:    deps.Hostname,
		ResolvePath: config.ResolveFile,
	}
	mapping := make(map[string]secretsync.MappingEntry, len(cfg.Mapping))
	for name, entry := range cfg.Mapping {
		mapping[name] = secretsync.MappingEntryFromConfig(entry)
	}
	return commandService{
		cfg:      cfg,
		api:      api,
		now:      syncDeps.Now,
		hostname: syncDeps.Hostname,
		inner: secretsync.New(secretsync.Config{
			Root:    cfg.Root,
			Mapping: mapping,
		}, api, syncDeps),
	}
}

func (s commandService) list(query listQuery) ([]listRecord, error) {
	return s.inner.List(query)
}

func (s commandService) lookupMappedSecret(name string, entry config.MappingEntry) (*secretprovider.SecretRecord, error) {
	return s.inner.LookupMappedSecret(name, secretsync.MappingEntryFromConfig(entry))
}

func (s commandService) resolveMappedSecret(name string, entry config.MappingEntry, createMissing bool) (*secretprovider.SecretRecord, error) {
	return s.inner.ResolveMappedSecret(name, secretsync.MappingEntryFromConfig(entry), createMissing)
}

func selectMappingTargets(mapping map[string]config.MappingEntry, all bool, positional []string, mode string) ([]string, error) {
	var typedMode commandMode
	switch mode {
	case "pull":
		typedMode = commandModePull
	case "push":
		typedMode = commandModePush
	default:
		typedMode = commandMode(0)
	}
	targets, err := selectMappingTargetsForMode(mapping, all, positional, typedMode)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names, nil
}

func parseSecretType(s string) (secretprovider.SecretType, error) {
	return secretsync.ParseSecretType(s)
}
