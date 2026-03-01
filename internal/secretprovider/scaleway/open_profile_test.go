package scaleway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestOpen_ProfileResolution(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "nope",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("Env", func(t *testing.T) {
		t.Setenv("SCW_ACCESS_KEY", "SCW1234567890ABCDEFG")                 // gitleaks:allow
		t.Setenv("SCW_SECRET_KEY", "00000000-0000-0000-0000-000000000000") // gitleaks:allow
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
		}, "")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("Env_NewClientError", func(t *testing.T) {
		t.Setenv("SCW_ACCESS_KEY", "SCW1234567890ABCDEFG")                 // gitleaks:allow
		t.Setenv("SCW_SECRET_KEY", "00000000-0000-0000-0000-000000000000") // gitleaks:allow
		_, err := Open(config.Config{
			OrganizationID: "not-a-uuid",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "create scaleway client") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ProfileMissingConfig", func(t *testing.T) {
		t.Setenv("SCW_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "p1",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("GetProfileError", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles: {}
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "missing",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "get scaleway profile") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Profile", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles:
  p1:
    access_key: SCW234567890ABCDEFGH # gitleaks:allow
    secret_key: 22222222-2222-2222-2222-222222222222 # gitleaks:allow
    default_organization_id: 22222222-2222-2222-2222-222222222222
    default_project_id: 22222222-2222-2222-2222-222222222222
    default_region: fr-par
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "p1",
		}, "")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("ProfileOverrideWins", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles:
  p2:
    access_key: SCW234567890ABCDEFGH # gitleaks:allow
    secret_key: 22222222-2222-2222-2222-222222222222 # gitleaks:allow
    default_organization_id: 22222222-2222-2222-2222-222222222222
    default_project_id: 22222222-2222-2222-2222-222222222222
    default_region: fr-par
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := Open(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "missing",
		}, "p2")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
}
