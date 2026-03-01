package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/fsx"
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

func (s commandService) list(query listQuery) ([]listRecord, error) {
	req := secretprovider.ListSecretsInput{
		Region:    s.cfg.Region,
		ProjectID: s.cfg.ProjectID,
	}
	if query.Path != "" {
		req.Path = query.Path
	}

	secretTypes := supportedSecretTypes()
	if query.Type != "" {
		st, err := parseSecretType(query.Type)
		if err != nil {
			return nil, usageError(fmt.Errorf("invalid --type: %w", err))
		}
		secretTypes = []secretprovider.SecretType{st}
	}

	respSecrets, err := listSecretsByTypes(s.api, req, secretTypes)
	if err != nil {
		return nil, runtimeError(err)
	}

	filtered := make([]listRecord, 0, len(respSecrets))
	for _, secretRecord := range respSecrets {
		if !strings.HasSuffix(secretRecord.Name, "-dev") {
			continue
		}
		if query.Path != "" && secretRecord.Path != query.Path {
			continue
		}
		if len(query.NameContains) > 0 {
			miss := false
			for _, c := range query.NameContains {
				if !strings.Contains(secretRecord.Name, c) {
					miss = true
					break
				}
			}
			if miss {
				continue
			}
		}
		if query.NameRegex != nil && !query.NameRegex.MatchString(secretRecord.Name) {
			continue
		}
		filtered = append(filtered, listRecord{
			ID:   secretRecord.ID,
			Name: secretRecord.Name,
			Path: secretRecord.Path,
			Type: string(secretRecord.Type),
		})
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered, nil
}

func (s commandService) pull(targets []mappingTarget, overwrite bool) ([]pullResult, error) {
	results := make([]pullResult, 0, len(targets))
	for _, target := range targets {
		outPath, err := config.ResolveFile(s.cfg.Root, target.Entry.File)
		if err != nil {
			return nil, fmt.Errorf("mapping %s: resolve file: %w", target.Name, err)
		}

		resolvedSecret, err := resolveSecretByNameAndPath(s.api, s.projectScope(), target.Name, target.Entry.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", target.Name, err)
		}
		if target.Entry.Type != "" && resolvedSecret.Type != secretprovider.SecretType(target.Entry.Type) {
			return nil, fmt.Errorf("secret %s: type mismatch (expected %s got %s)", target.Name, target.Entry.Type, resolvedSecret.Type)
		}

		access, err := s.api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{
			Region:   s.cfg.Region,
			SecretID: resolvedSecret.ID,
			Revision: secretprovider.SecretRevisionLatestEnabled,
		})
		if err != nil {
			return nil, fmt.Errorf("access %s: %w", target.Name, err)
		}

		payload := access.Data
		if target.Entry.Format == "dotenv" {
			converted, err := jsonToDotenv(payload)
			if err != nil {
				return nil, fmt.Errorf("format dotenv %s: %w", target.Name, err)
			}
			payload = converted
		}

		if err := fsx.AtomicWriteFile(outPath, payload, 0o600, overwrite); err != nil {
			if errors.Is(err, fsx.ErrExists) {
				return nil, fmt.Errorf("pull %s: file exists (use --overwrite): %s", target.Name, outPath)
			}
			return nil, fmt.Errorf("pull %s: write %s: %w", target.Name, outPath, err)
		}

		results = append(results, pullResult{
			Name:     target.Name,
			File:     target.Entry.File,
			Revision: access.Revision,
			Type:     string(access.Type),
		})
	}
	return results, nil
}

func (s commandService) push(targets []mappingTarget, opts pushOptions) ([]pushResult, error) {
	desc := s.pushDescription(opts.Description)

	results := make([]pushResult, 0, len(targets))
	for _, target := range targets {
		payload, err := s.readPushPayload(target.Name, target.Entry)
		if err != nil {
			return nil, err
		}
		resolvedSecret, err := s.resolvePushSecret(target.Name, target.Entry, opts.CreateMissing)
		if err != nil {
			return nil, err
		}

		disablePrevPtr := (*bool)(nil)
		if opts.DisablePrevious {
			disablePrevPtr = new(bool)
			*disablePrevPtr = true
		}
		descPtr := &desc

		version, err := s.api.CreateSecretVersion(secretprovider.CreateSecretVersionInput{
			Region:          s.cfg.Region,
			SecretID:        resolvedSecret.ID,
			Data:            payload,
			Description:     descPtr,
			DisablePrevious: disablePrevPtr,
		})
		if err != nil {
			return nil, fmt.Errorf("push %s: create version: %w", target.Name, err)
		}

		results = append(results, pushResult{Name: target.Name, Revision: version.Revision})
	}

	return results, nil
}

func (s commandService) pushDescription(explicit string) string {
	if explicit != "" {
		return explicit
	}
	host := "unknown-host"
	if h, err := s.hostname(); err == nil && h != "" {
		host = h
	}
	return fmt.Sprintf("dev-vault push %s %s", s.now().UTC().Format(time.RFC3339), host)
}

func (s commandService) readPushPayload(name string, entry config.MappingEntry) ([]byte, error) {
	inPath, err := config.ResolveFile(s.cfg.Root, entry.File)
	if err != nil {
		return nil, fmt.Errorf("mapping %s: resolve file: %w", name, err)
	}
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return nil, fmt.Errorf("push %s: read %s: %w", name, inPath, err)
	}
	if entry.Format == "dotenv" {
		converted, err := dotenvToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("format dotenv %s: %w", name, err)
		}
		return converted, nil
	}
	return raw, nil
}

func (s commandService) resolvePushSecret(name string, entry config.MappingEntry, createMissing bool) (*secretprovider.SecretRecord, error) {
	resolvedSecret, err := resolveSecretByNameAndPath(s.api, s.projectScope(), name, entry.Path)
	if err == nil {
		if entry.Type != "" && resolvedSecret.Type != secretprovider.SecretType(entry.Type) {
			return nil, fmt.Errorf("secret %s: type mismatch (expected %s got %s)", name, entry.Type, resolvedSecret.Type)
		}
		return resolvedSecret, nil
	}

	var notFound *notFoundError
	if !errors.As(err, &notFound) || !createMissing {
		return nil, fmt.Errorf("resolve %s: %w", name, err)
	}
	if entry.Type == "" {
		return nil, fmt.Errorf("push %s: create-missing requires mapping.type", name)
	}

	secretType, err := parseSecretType(entry.Type)
	if err != nil {
		return nil, fmt.Errorf("push %s: invalid mapping.type %q: %w", name, entry.Type, err)
	}
	createdSecret, err := s.api.CreateSecret(secretprovider.CreateSecretInput{
		Region:    s.cfg.Region,
		ProjectID: s.cfg.ProjectID,
		Name:      name,
		Type:      secretType,
		Path:      entry.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("push %s: create secret: %w", name, err)
	}
	return createdSecret, nil
}
