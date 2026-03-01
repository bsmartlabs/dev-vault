//go:build integration

package contracts_test

import (
	"os"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	providercontracts "github.com/bsmartlabs/dev-vault/internal/secretprovider/contracts"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider/scaleway"
)

func TestScalewayProviderReadOnlyContracts(t *testing.T) {
	projectID := os.Getenv("DEV_VAULT_TEST_PROJECT_ID")
	orgID := os.Getenv("DEV_VAULT_TEST_ORGANIZATION_ID")
	region := os.Getenv("DEV_VAULT_TEST_REGION")
	if region == "" {
		region = "fr-par"
	}
	if projectID == "" || orgID == "" {
		t.Skip("set DEV_VAULT_TEST_PROJECT_ID and DEV_VAULT_TEST_ORGANIZATION_ID to run integration secret API gate")
	}

	err := providercontracts.RunReadOnlyListSuiteWithOpen(func() (secretprovider.SecretAPI, error) {
		return scaleway.Open(config.Config{
			OrganizationID: orgID,
			ProjectID:      projectID,
			Region:         region,
		}, "")
	}, "/", []secretprovider.SecretType{
		secretprovider.SecretTypeOpaque,
		secretprovider.SecretTypeKeyValue,
	})
	if err != nil {
		t.Fatalf("provider read-only contract: %v", err)
	}
}
