package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretworkflow"
)

func (s commandService) push(targets []mappingTarget, opts pushOptions) ([]pushResult, error) {
	desc := s.pushDescription(opts.Description)

	results := make([]pushResult, 0, len(targets))
	for _, target := range targets {
		payload, err := s.readPushPayload(target.Name, target.Entry)
		if err != nil {
			return nil, err
		}
		resolvedSecret, err := s.resolveMappedSecret(target.Name, target.Entry, opts.CreateMissing)
		if err != nil {
			return nil, err
		}

		version, err := s.api.CreateSecretVersion(createSecretVersionInput(
			resolvedSecret.ID,
			payload,
			desc,
			opts.DisablePrevious,
		))
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
		converted, err := secretworkflow.DotenvToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("format dotenv %s: %w", name, err)
		}
		return converted, nil
	}
	return raw, nil
}

func createSecretVersionInput(secretID string, payload []byte, description string, disablePrevious bool) secretprovider.CreateSecretVersionInput {
	req := secretprovider.CreateSecretVersionInput{
		SecretID:    secretID,
		Data:        payload,
		Description: &description,
	}
	if disablePrevious {
		disablePreviousValue := true
		req.DisablePrevious = &disablePreviousValue
	}
	return req
}

func (s commandService) resolveMappedSecret(name string, entry config.MappingEntry, createMissing bool) (*secretprovider.SecretRecord, error) {
	resolvedSecret, err := s.lookupMappedSecret(name, entry)
	if err == nil {
		return resolvedSecret, nil
	}

	var notFound *secretLookupMissError
	if !errors.As(err, &notFound) || !createMissing {
		return nil, fmt.Errorf("resolve %s: %w", name, err)
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
