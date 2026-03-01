package secretsync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type SecretLookupMissError struct {
	Name string
	Path string
}

func (e *SecretLookupMissError) Error() string {
	return fmt.Sprintf("secret not found: name=%s path=%s", e.Name, e.Path)
}

func (s Service) LookupMappedSecret(name string, entry MappingEntry) (*secretprovider.SecretRecord, error) {
	return s.lookupMappedSecret(name, entry)
}

func (s Service) lookupMappedSecret(name string, entry MappingEntry) (*secretprovider.SecretRecord, error) {
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
