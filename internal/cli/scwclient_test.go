package cli

import (
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestOpenScalewaySecretAPI_InvalidRegionSmoke(t *testing.T) {
	_, err := OpenScalewaySecretAPI(config.Config{
		OrganizationID: "00000000-0000-0000-0000-000000000000",
		ProjectID:      "00000000-0000-0000-0000-000000000000",
		Region:         "invalid-region",
	}, "")
	if err == nil {
		t.Fatalf("expected error")
	}
}
