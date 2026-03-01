package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/fsx"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type commandService struct {
	loaded   *config.Loaded
	api      SecretAPI
	now      func() time.Time
	hostname func() (string, error)
}

type pullResult struct {
	Name     string
	File     string
	Revision uint32
	Type     secret.SecretType
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
		api:      api,
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

		resolvedSecret, err := resolveSecretByNameAndPath(s.api, s.loaded.Cfg, name, entry.Path)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", name, err)
		}
		if entry.Type != "" && string(resolvedSecret.Type) != entry.Type {
			return nil, fmt.Errorf("secret %s: type mismatch (expected %s got %s)", name, entry.Type, resolvedSecret.Type)
		}

		access, err := s.api.AccessSecretVersion(&secret.AccessSecretVersionRequest{
			Region:   scw.Region(s.loaded.Cfg.Region),
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
	desc := opts.Description
	if desc == "" {
		host := "unknown-host"
		if h, err := s.hostname(); err == nil && h != "" {
			host = h
		}
		desc = fmt.Sprintf("dev-vault push %s %s", s.now().UTC().Format(time.RFC3339), host)
	}

	results := make([]pushResult, 0, len(targets))
	for _, name := range targets {
		entry := s.loaded.Cfg.Mapping[name]
		inPath, err := config.ResolveFile(s.loaded.Root, entry.File)
		if err != nil {
			return nil, fmt.Errorf("mapping %s: resolve file: %w", name, err)
		}
		raw, err := os.ReadFile(inPath)
		if err != nil {
			return nil, fmt.Errorf("push %s: read %s: %w", name, inPath, err)
		}

		payload := raw
		if entry.Format == "dotenv" {
			converted, err := dotenvToJSON(payload)
			if err != nil {
				return nil, fmt.Errorf("format dotenv %s: %w", name, err)
			}
			payload = converted
		}

		resolvedSecret, err := resolveSecretByNameAndPath(s.api, s.loaded.Cfg, name, entry.Path)
		if err != nil {
			var notFound *notFoundError
			if errors.As(err, &notFound) && opts.CreateMissing {
				if entry.Type == "" {
					return nil, fmt.Errorf("push %s: create-missing requires mapping.type", name)
				}
				secretType := mustParseSecretType(entry.Type)
				_, err = s.api.CreateSecret(&secret.CreateSecretRequest{
					Region:      scw.Region(s.loaded.Cfg.Region),
					ProjectID:   s.loaded.Cfg.ProjectID,
					Name:        name,
					Tags:        []string{},
					Description: nil,
					Type:        secretType,
					Path:        scw.StringPtr(entry.Path),
					Protected:   false,
					KeyID:       nil,
				})
				if err != nil {
					return nil, fmt.Errorf("push %s: create secret: %w", name, err)
				}
				resolvedSecret, err = resolveSecretByNameAndPath(s.api, s.loaded.Cfg, name, entry.Path)
				if err != nil {
					return nil, fmt.Errorf("push %s: after create, resolve failed: %w", name, err)
				}
			} else {
				return nil, fmt.Errorf("resolve %s: %w", name, err)
			}
		}

		if entry.Type != "" && string(resolvedSecret.Type) != entry.Type {
			return nil, fmt.Errorf("secret %s: type mismatch (expected %s got %s)", name, entry.Type, resolvedSecret.Type)
		}

		disablePrevPtr := (*bool)(nil)
		if opts.DisablePrevious {
			disablePrevPtr = scw.BoolPtr(true)
		}
		descPtr := scw.StringPtr(desc)

		version, err := s.api.CreateSecretVersion(&secret.CreateSecretVersionRequest{
			Region:          scw.Region(s.loaded.Cfg.Region),
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
