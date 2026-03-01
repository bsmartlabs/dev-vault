package cli

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/fsx"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func (s commandService) pull(targets []mappingTarget, overwrite bool) ([]pullResult, error) {
	lookupIndex, err := buildSecretLookupIndex(s.api, s.projectScope())
	if err != nil {
		return nil, fmt.Errorf("build secret lookup index: %w", err)
	}

	results := make([]pullResult, 0, len(targets))
	for _, target := range targets {
		outPath, err := config.ResolveFile(s.cfg.Root, target.Entry.File)
		if err != nil {
			return nil, fmt.Errorf("mapping %s: resolve file: %w", target.Name, err)
		}

		resolvedSecret, err := s.resolveMappedSecret(target.Name, target.Entry, false, lookupIndex)
		if err != nil {
			return nil, err
		}

		access, err := s.api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{
			Region:   s.cfg.Region,
			SecretID: resolvedSecret.ID,
			Revision: secretprovider.RevisionLatestEnabled,
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
