package secretsync

import (
	"fmt"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretworkflow"
)

func (s Service) PushBatch(targets []mapping.Target, opts PushOptions) PushBatchResult {
	desc := s.pushDescription(opts.Description)
	succeeded, failed, summary := runBatch[PushResult](
		targets,
		"push",
		func(target mapping.Target) (PushResult, error) {
			return s.pushOne(target, opts, desc)
		},
		func(target mapping.Target, err error) BatchFailure {
			return BatchFailure{Name: target.Name, Err: err}
		},
	)
	result := PushBatchResult{Succeeded: succeeded, Failed: failed, Summary: summary}
	return result
}

func (s Service) pushDescription(explicit string) string {
	if explicit != "" {
		return explicit
	}
	host := "unknown-host"
	if h, err := s.hostname(); err == nil && h != "" {
		host = h
	}
	return fmt.Sprintf("dev-vault push %s %s", s.now().UTC().Format(time.RFC3339), host)
}

func (s Service) readPushPayload(name string, entry mapping.Entry) ([]byte, error) {
	inPath, err := s.resolvePath(s.cfg.Root, entry.File)
	if err != nil {
		return nil, fmt.Errorf("mapping %s: resolve file: %w", name, err)
	}
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return nil, fmt.Errorf("push %s: read %s: %w", name, inPath, err)
	}
	if entry.Format == mapping.FormatDotenv {
		converted, err := secretworkflow.DotenvToJSON(raw)
		if err != nil {
			return nil, fmt.Errorf("format dotenv %s: %w", name, err)
		}
		return converted, nil
	}
	return raw, nil
}

func (s Service) pushOne(target mapping.Target, opts PushOptions, desc string) (PushResult, error) {
	payload, err := s.readPushPayload(target.Name, target.Entry)
	if err != nil {
		return PushResult{}, err
	}

	var resolvedSecret *secretprovider.SecretRecord
	if opts.CreateMissing {
		resolvedSecret, err = s.lookupOrCreateMappedSecret(target.Name, target.Entry)
	} else {
		resolvedSecret, err = s.lookupMappedSecret(target.Name, target.Entry)
	}
	if err != nil {
		return PushResult{}, fmt.Errorf("resolve %s: %w", target.Name, err)
	}

	req := secretprovider.CreateSecretVersionInput{
		SecretID:    resolvedSecret.ID,
		Data:        payload,
		Description: &desc,
	}
	if opts.DisablePrevious {
		dp := true
		req.DisablePrevious = &dp
	}
	version, err := s.api.CreateSecretVersion(req)
	if err != nil {
		return PushResult{}, fmt.Errorf("push %s: create version: %w", target.Name, err)
	}
	return PushResult{Name: target.Name, Revision: version.Revision}, nil
}
