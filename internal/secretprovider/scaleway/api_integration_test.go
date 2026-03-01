//go:build integration

package scaleway

import (
	"os"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func TestScalewaySecretAPI_IntegrationListOpaque(t *testing.T) {
	projectID := os.Getenv("DEV_VAULT_TEST_PROJECT_ID")
	orgID := os.Getenv("DEV_VAULT_TEST_ORGANIZATION_ID")
	region := os.Getenv("DEV_VAULT_TEST_REGION")
	if region == "" {
		region = "fr-par"
	}
	if projectID == "" || orgID == "" {
		t.Skip("set DEV_VAULT_TEST_PROJECT_ID and DEV_VAULT_TEST_ORGANIZATION_ID to run integration secret API gate")
	}

	api, err := Open(config.Config{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Region:         region,
	}, "")
	if err != nil {
		t.Fatalf("open scaleway api: %v", err)
	}

	_, err = api.ListSecrets(secretprovider.ListSecretsInput{
		Region:    region,
		ProjectID: projectID,
		Path:      "/",
		Type:      secretprovider.SecretTypeOpaque,
	})
	if err != nil {
		t.Fatalf("list secrets via secret api: %v", err)
	}
}
