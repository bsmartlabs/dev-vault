package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type commandServiceConfig struct {
	Root    string
	Mapping map[string]mapping.Entry
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
		ResolvePath: deps.ResolveProjectPath,
	}
	inner, err := secretsync.New(secretsync.Config{
		Root: cfg.Root,
	}, api, syncDeps)
	if err != nil {
		panic(err)
	}
	return commandService{
		cfg:      cfg,
		api:      api,
		now:      syncDeps.Now,
		hostname: syncDeps.Hostname,
		inner:    inner,
	}
}

func (s commandService) list(query listQuery) ([]listRecord, error) {
	return s.inner.List(query)
}

func (s commandService) lookupMappedSecret(name string, entry mapping.Entry) (*secretprovider.SecretRecord, error) {
	req := secretprovider.ListSecretsInput{
		Name: name,
		Path: entry.Path,
		Type: secretprovider.SecretType(entry.Type),
	}
	respSecrets, err := s.api.ListSecrets(req)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	matches := make([]secretprovider.SecretRecord, 0, len(respSecrets))
	for _, secretRecord := range respSecrets {
		if secretRecord.Name == name && secretRecord.Path == entry.Path {
			matches = append(matches, secretRecord)
		}
	}
	if len(matches) == 0 {
		return nil, &secretLookupMissError{Name: name, Path: entry.Path}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, secretRecord := range matches {
			ids = append(ids, secretRecord.ID)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("multiple secrets match name=%s path=%s: %s", name, entry.Path, strings.Join(ids, ","))
	}
	resolved := matches[0]
	return &resolved, nil
}

func (s commandService) resolveMappedSecret(name string, entry mapping.Entry, createMissing bool) (*secretprovider.SecretRecord, error) {
	resolvedSecret, err := s.lookupMappedSecret(name, entry)
	if err == nil {
		return resolvedSecret, nil
	}
	var notFound *secretLookupMissError
	if !createMissing || !errors.As(err, &notFound) {
		if createMissing {
			return nil, fmt.Errorf("resolve %s: %w", name, err)
		}
		return nil, err
	}
	if entry.Type == "" {
		return nil, fmt.Errorf("push %s: create-missing requires mapping.type", name)
	}
	createdSecret, err := s.api.CreateSecret(secretprovider.CreateSecretInput{
		Name: name,
		Type: secretprovider.SecretType(entry.Type),
		Path: entry.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("push %s: create secret: %w", name, err)
	}
	return createdSecret, nil
}

func selectMappingTargets(entries map[string]mapping.Entry, all bool, positional []string, mode string) ([]string, error) {
	var typedMode mapping.Mode
	switch mode {
	case "pull":
		typedMode = mapping.ModePull
	case "push":
		typedMode = mapping.ModePush
	default:
		typedMode = mapping.Mode("")
	}
	targets, err := mapping.SelectTargetsForMode(entries, all, positional, typedMode)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names, nil
}
