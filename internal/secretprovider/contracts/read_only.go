package contracts

import (
	"fmt"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type OpenProviderFunc func() (secretprovider.SecretAPI, error)

func RunReadOnlyListSuiteWithOpen(open OpenProviderFunc, path string, types []secretprovider.SecretType) error {
	api, err := open()
	if err != nil {
		return fmt.Errorf("open provider for contracts: %w", err)
	}
	return RunReadOnlyListSuite(api, path, types)
}

// RunReadOnlyListSuite validates read-only ListSecrets conformance at the
// secretprovider boundary without mutating remote state.
func RunReadOnlyListSuite(api secretprovider.SecretAPI, path string, types []secretprovider.SecretType) error {
	if path == "" {
		path = "/"
	}

	for _, secretType := range types {
		if _, err := api.ListSecrets(secretprovider.ListSecretsInput{Path: path, Type: secretType}); err != nil {
			return fmt.Errorf("list with type %q failed: %w", secretType, err)
		}
	}

	const impossibleName = "dev-vault-contract-this-name-should-not-exist-dev"
	if _, err := api.ListSecrets(secretprovider.ListSecretsInput{Path: path, Name: impossibleName}); err != nil {
		return fmt.Errorf("list with name filter failed: %w", err)
	}

	_, err := api.ListSecrets(secretprovider.ListSecretsInput{Type: secretprovider.SecretType("not-a-secret-type")})
	if err == nil {
		return fmt.Errorf("invalid type did not fail validation")
	}
	if !strings.Contains(err.Error(), "invalid secret type") {
		return fmt.Errorf("invalid type error shape mismatch: %w", err)
	}

	return nil
}
