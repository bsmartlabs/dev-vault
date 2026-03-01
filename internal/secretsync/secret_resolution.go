package secretsync

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type SecretLookupMissError struct {
	Name string
	Path string
}

func (e *SecretLookupMissError) Error() string {
	return fmt.Sprintf("secret not found: name=%s path=%s", e.Name, e.Path)
}

func (s Service) lookupOrCreateMappedSecret(name string, entry mapping.Entry) (*secretprovider.SecretRecord, error) {
	resolvedSecret, err := s.lookupMappedSecret(name, entry)
	if err == nil {
		return resolvedSecret, nil
	}

	var notFound *SecretLookupMissError
	if !errors.As(err, &notFound) {
		return nil, err
	}
	if entry.Type == "" {
		return nil, errors.New("create-missing requires mapping.type")
	}

	createdSecret, err := s.api.CreateSecret(secretprovider.CreateSecretInput{
		Name: name,
		Type: secretprovider.SecretType(entry.Type),
		Path: entry.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}
	return createdSecret, nil
}

func (s Service) lookupMappedSecret(name string, entry mapping.Entry) (*secretprovider.SecretRecord, error) {
	req := secretprovider.ListSecretsInput{
		Name: name,
		Path: entry.Path,
	}

	if entry.Type != "" {
		req.Type = secretprovider.SecretType(entry.Type)
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
		return nil, &SecretLookupMissError{Name: name, Path: entry.Path}
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
