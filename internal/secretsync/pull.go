package secretsync

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/fsx"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretworkflow"
)

func (s Service) PullBatch(targets []MappingTarget, overwrite bool) PullBatchResult {
	succeeded, failed, summary := runBatch[PullResult](
		targets,
		"pull",
		func(target MappingTarget) (PullResult, error) {
			return s.pullOne(target, overwrite)
		},
		func(target MappingTarget, err error) BatchFailure {
			return BatchFailure{Name: target.Name, Err: err}
		},
	)
	return PullBatchResult{
		Succeeded: succeeded,
		Failed:    failed,
		Summary:   summary,
	}
}

func (s Service) pullOne(target MappingTarget, overwrite bool) (PullResult, error) {
	outPath, err := s.resolvePath(s.cfg.Root, target.Entry.File)
	if err != nil {
		return PullResult{}, fmt.Errorf("mapping %s: resolve file: %w", target.Name, err)
	}

	resolvedSecret, err := s.lookupMappedSecret(target.Name, target.Entry)
	if err != nil {
		return PullResult{}, fmt.Errorf("resolve %s: %w", target.Name, err)
	}

	access, err := s.api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{
		SecretID: resolvedSecret.ID,
		Revision: secretprovider.RevisionLatestEnabled,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("access %s: %w", target.Name, err)
	}

	payload := access.Data
	if target.Entry.Format == mapping.FormatDotenv {
		converted, err := secretworkflow.JSONToDotenv(payload)
		if err != nil {
			return PullResult{}, fmt.Errorf("format dotenv %s: %w", target.Name, err)
		}
		payload = converted
	}

	if err := fsx.AtomicWriteFile(outPath, payload, 0o600, overwrite); err != nil {
		if errors.Is(err, fsx.ErrExists) {
			return PullResult{}, fmt.Errorf("pull %s: file exists (use --overwrite): %s", target.Name, outPath)
		}
		return PullResult{}, fmt.Errorf("pull %s: write %s: %w", target.Name, outPath, err)
	}

	return PullResult{
		Name:     target.Name,
		File:     target.Entry.File,
		Revision: access.Revision,
		Type:     access.Type,
	}, nil
}
