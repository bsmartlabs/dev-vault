package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

func supportedSecretTypes() []secretprovider.SecretType {
	names := secrettype.Names()
	out := make([]secretprovider.SecretType, 0, len(names))
	for _, name := range names {
		out = append(out, secretprovider.SecretType(name))
	}
	return out
}

type notFoundError struct {
	name string
	path string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("secret not found: name=%s path=%s", e.name, e.path)
}

type secretProjectScope struct {
	Region    string
	ProjectID string
}

func resolveSecretByNameAndPath(api secretprovider.SecretLister, scope secretProjectScope, name, path string) (*secretprovider.SecretRecord, error) {
	respSecrets, err := listSecretsByTypes(api, secretprovider.ListSecretsInput{
		Region:    scope.Region,
		ProjectID: scope.ProjectID,
		Name:      name,
		Path:      path,
	}, supportedSecretTypes())
	if err != nil {
		return nil, err
	}

	matches := make([]secretprovider.SecretRecord, 0, len(respSecrets))
	for _, s := range respSecrets {
		if s.Name == name && s.Path == path {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, &notFoundError{name: name, path: path}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, s := range matches {
			ids = append(ids, s.ID)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("multiple secrets match name=%s path=%s: %s", name, path, strings.Join(ids, ","))
	}
	return &matches[0], nil
}

func listSecretsByTypes(api secretprovider.SecretLister, base secretprovider.ListSecretsInput, types []secretprovider.SecretType) ([]secretprovider.SecretRecord, error) {
	var out []secretprovider.SecretRecord
	for _, t := range types {
		req := base
		req.Type = t
		resp, err := api.ListSecrets(req)
		if err != nil {
			return nil, fmt.Errorf("list secrets for type=%s: %w", t, err)
		}
		out = append(out, resp...)
	}
	return out, nil
}
