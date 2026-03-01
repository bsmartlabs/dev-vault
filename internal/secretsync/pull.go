package secretsync

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/fsx"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretworkflow"
)

func (s Service) Pull(targets []MappingTarget, overwrite bool) ([]PullResult, error) {
	results := make([]PullResult, 0, len(targets))
	for _, target := range targets {
		outPath, err := s.resolvePath(s.cfg.Root, target.Entry.File)
		if err != nil {
			return nil, fmt.Errorf("mapping %s: resolve file: %w", target.Name, err)
		}

		resolvedSecret, err := s.lookupMappedSecret(target.Name, target.Entry)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", target.Name, err)
		}

		access, err := s.api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{
			SecretID: resolvedSecret.ID,
			Revision: secretprovider.RevisionLatestEnabled,
		})
		if err != nil {
			return nil, fmt.Errorf("access %s: %w", target.Name, err)
		}

		payload := access.Data
		if target.Entry.Format == MappingFormatDotenv {
			converted, err := secretworkflow.JSONToDotenv(payload)
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

		results = append(results, PullResult{
			Name:     target.Name,
			File:     target.Entry.File,
			Revision: access.Revision,
			Type:     string(access.Type),
		})
	}
	return results, nil
}
