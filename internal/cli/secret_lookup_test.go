package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

func resolveSecretByNameAndPathFromIndex(api secretprovider.SecretLister, scope secretProjectScope, name, path string) (*secretprovider.SecretRecord, error) {
	index, err := buildSecretLookupIndex(api, scope)
	if err != nil {
		return nil, err
	}
	return resolveSecretFromIndex(index, name, path)
}

func TestSecretLookupFile_BasicSmoke(t *testing.T) {
	msg := (&notFoundError{name: "x-dev", path: "/"}).Error()
	if !strings.Contains(msg, "x-dev") {
		t.Fatalf("unexpected notFoundError message: %q", msg)
	}

	fake := newFakeSecretAPI()
	s := fake.AddSecret("project", "x-dev", "/", secret.SecretTypeOpaque)

	found, err := resolveSecretByNameAndPathFromIndex(fake, secretProjectScope{
		Region:    "fr-par",
		ProjectID: "project",
	}, "x-dev", "/")
	if err != nil {
		t.Fatalf("resolveSecretByNameAndPathFromIndex: %v", err)
	}
	if found.ID != s.ID {
		t.Fatalf("resolved wrong secret id: got %q want %q", found.ID, s.ID)
	}

	secrets, err := listSecretsByTypes(fake, ListSecretsInput{
		ProjectID: s.ProjectID,
		Path:      s.Path,
	}, []SecretType{SecretTypeOpaque})
	if err != nil {
		t.Fatalf("listSecretsByTypes: %v", err)
	}
	if len(secrets) != 1 || secrets[0].ID != s.ID {
		t.Fatalf("unexpected secrets list: %#v", secrets)
	}
}

func TestResolveSecretByNameAndPath_ListError(t *testing.T) {
	fake := newFakeSecretAPI()
	fake.listErr = errors.New("boom")
	_, err := resolveSecretByNameAndPathFromIndex(fake, secretProjectScope{
		Region:    "fr-par",
		ProjectID: "project",
	}, "x-dev", "/")
	if err == nil {
		t.Fatal("expected list error")
	}
}
