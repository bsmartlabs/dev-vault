package cli

import (
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

func TestSecretLookupFile_BasicSmoke(t *testing.T) {
	msg := (&notFoundError{name: "x-dev", path: "/"}).Error()
	if !strings.Contains(msg, "x-dev") {
		t.Fatalf("unexpected notFoundError message: %q", msg)
	}

	fake := newFakeSecretAPI()
	s := fake.AddSecret("project", "x-dev", "/", secret.SecretTypeOpaque)

	found, err := resolveSecretByNameAndPath(fake, config.Config{
		Region:    "fr-par",
		ProjectID: "project",
	}, "x-dev", "/")
	if err != nil {
		t.Fatalf("resolveSecretByNameAndPath: %v", err)
	}
	if found.ID != s.ID {
		t.Fatalf("resolved wrong secret id: got %q want %q", found.ID, s.ID)
	}

	secrets, err := listSecretsByTypes(fake, ListSecretsInput{
		ProjectID: s.ProjectID,
		Path:      s.Path,
	}, []string{string(secret.SecretTypeOpaque)})
	if err != nil {
		t.Fatalf("listSecretsByTypes: %v", err)
	}
	if len(secrets) != 1 || secrets[0].ID != s.ID {
		t.Fatalf("unexpected secrets list: %#v", secrets)
	}
}
