package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/fsx"
)

type commandService struct {
	loaded   *config.Loaded
	lister   SecretLister
	accessor SecretVersionAccessor
	creator  SecretCreator
	version  SecretVersionCreator
	now      func() time.Time
	hostname func() (string, error)
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

func newCommandService(loaded *config.Loaded, api SecretAPI, deps Dependencies) commandService {
	return commandService{
		loaded:   loaded,
		lister:   api,
		accessor: api,
		creator:  api,
		version:  api,
		now:      deps.Now,
		hostname: deps.Hostname,
	}
}

func (s commandService) pull(targets []string, overwrite bool) ([]pullResult, error) {
	results := make([]pullResult, 0, len(targets))
	for _, name := range targets {
		entry := s.loaded.Cfg.Mapping[name]
		outPath, err := config.ResolveFile(s.loaded.Root, entry.File)
		if err != nil {
			return nil, fmt.Errorf("mapping %s: resolve file: %w", name, err)
		}

		resolvedSecret, err := resolveSecretByNameAndPath(s.lister, s.loaded.Cfg, name, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", name, err)
		}
		if entry.Type != "" && resolvedSecret.Type != entry.Type {
			return nil, fmt.Errorf("secret %s: type mismatch (expected %s got %s)", name, entry.Type, resolvedSecret.Type)
		}

		access, err := s.accessor.AccessSecretVersion(AccessSecretVersionInput{
			Region:   s.loaded.Cfg.Region,
			SecretID: resolvedSecret.ID,
			Revision: "latest_enabled",
		})
		if err != nil {
			return nil, fmt.Errorf("access %s: %w", name, err)
		}

		payload := access.Data
		if entry.Format == "dotenv" {
			converted, err := jsonToDotenv(payload)
			if err != nil {
				return nil, fmt.Errorf("format dotenv %s: %w", name, err)
			}
			payload = converted
		}

		if err := fsx.AtomicWriteFile(outPath, payload, 0o600, overwrite); err != nil {
			if errors.Is(err, fsx.ErrExists) {
				return nil, fmt.Errorf("pull %s: file exists (use --overwrite): %s", name, outPath)
			}
			return nil, fmt.Errorf("pull %s: write %s: %w", name, outPath, err)
		}

		results = append(results, pullResult{
			Name:     name,
			File:     entry.File,
			Revision: access.Revision,
			Type:     access.Type,
		})
	}
	return results, nil
}

func (s commandService) push(targets []string, opts pushOptions) ([]pushResult, error) {
	desc := s.pushDescription(opts.Description)

	results := make([]pushResult, 0, len(targets))
	for _, name := range targets {
		entry := s.loaded.Cfg.Mapping[name]
		payload, err := s.readPushPayload(name, entry)
		if err != nil {
			return nil, err
		}
		resolvedSecret, err := s.resolvePushSecret(name, entry, opts.CreateMissing)
		if err != nil {
			return nil, err
		}

		disablePrevPtr := (*bool)(nil)
		if opts.DisablePrevious {
			disablePrevPtr = new(bool)
			*disablePrevPtr = true
		}
		descPtr := &desc

		version, err := s.version.CreateSecretVersion(CreateSecretVersionInput{
			Region:          s.loaded.Cfg.Region,
			SecretID:        resolvedSecret.ID,
			Data:            payload,
			Description:     descPtr,
			DisablePrevious: disablePrevPtr,
		})
		if err != nil {
			return nil, fmt.Errorf("push %s: create version: %w", name, err)
		}

		results = append(results, pushResult{Name: name, Revision: version.Revision})
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
	inPath, err := config.ResolveFile(s.loaded.Root, entry.File)
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

func (s commandService) resolvePushSecret(name string, entry config.MappingEntry, createMissing bool) (*SecretRecord, error) {
	resolvedSecret, err := resolveSecretByNameAndPath(s.lister, s.loaded.Cfg, name, entry.Path)
	if err == nil {
		if entry.Type != "" && resolvedSecret.Type != entry.Type {
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
	createdSecret, err := s.creator.CreateSecret(CreateSecretInput{
		Region:    s.loaded.Cfg.Region,
		ProjectID: s.loaded.Cfg.ProjectID,
		Name:      name,
		Type:      secretType,
		Path:      entry.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("push %s: create secret: %w", name, err)
	}
	return createdSecret, nil
}
