package secretapi

import (
	"errors"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func TestNewConstructors(t *testing.T) {
	seq := NewFakeSecretAPI()
	s1 := seq.AddSecret("proj", "a-dev", "/", secretprovider.SecretTypeOpaque)
	if s1.ID != "sec-1" {
		t.Fatalf("expected sequential ID, got %q", s1.ID)
	}

	det := NewDeterministicFakeSecretAPI()
	s2 := det.AddSecret("proj", "a-dev", "/", secretprovider.SecretTypeOpaque)
	if s2.ID != "sec-a-dev-proj" {
		t.Fatalf("expected deterministic ID, got %q", s2.ID)
	}
}

func TestListSecretsFiltersAndErrors(t *testing.T) {
	api := NewFakeSecretAPI()
	api.ListErr = errors.New("boom")
	if _, err := api.ListSecrets(secretprovider.ListSecretsInput{}); err == nil {
		t.Fatal("expected list error")
	}
	api.ListErr = nil

	api.AddSecret("proj", "a-dev", "/a", secretprovider.SecretTypeOpaque)
	api.AddSecret("proj", "b-dev", "/b", secretprovider.SecretTypeKeyValue)
	api.AddSecret("proj", "c-dev", "/c", secretprovider.SecretTypeCertificate)

	got, err := api.ListSecrets(secretprovider.ListSecretsInput{
		Name: "a-dev",
		Path: "/a",
		Type: secretprovider.SecretTypeOpaque,
	})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "a-dev" {
		t.Fatalf("unexpected list result: %#v", got)
	}
	if api.ListCalls == 0 {
		t.Fatal("expected list call counter increment")
	}

	if got, err := api.ListSecrets(secretprovider.ListSecretsInput{Name: "missing-dev"}); err != nil || len(got) != 0 {
		t.Fatalf("expected name filter miss, got %#v err=%v", got, err)
	}
	if got, err := api.ListSecrets(secretprovider.ListSecretsInput{Path: "/missing"}); err != nil || len(got) != 0 {
		t.Fatalf("expected path filter miss, got %#v err=%v", got, err)
	}
	if got, err := api.ListSecrets(secretprovider.ListSecretsInput{Type: secretprovider.SecretType("unknown")}); err != nil || len(got) != 0 {
		t.Fatalf("expected type filter miss, got %#v err=%v", got, err)
	}
}

func TestAccessSecretVersionBranches(t *testing.T) {
	api := NewFakeSecretAPI()
	secRec := api.AddSecret("proj", "a-dev", "/", secretprovider.SecretTypeOpaque)

	api.AccessErr = errors.New("boom")
	if _, err := api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{SecretID: secRec.ID, Revision: secretprovider.RevisionLatestEnabled}); err == nil {
		t.Fatal("expected access error")
	}
	api.AccessErr = nil

	if _, err := api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{SecretID: "missing", Revision: secretprovider.RevisionLatestEnabled}); err == nil {
		t.Fatal("expected unknown secret error")
	}
	if _, err := api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{SecretID: secRec.ID, Revision: "rev-1"}); err == nil {
		t.Fatal("expected unsupported revision selector")
	}
	if _, err := api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{SecretID: secRec.ID, Revision: secretprovider.RevisionLatestEnabled}); err == nil {
		t.Fatal("expected no enabled version error")
	}

	api.AddEnabledVersion(secRec.ID, []byte("one"))
	api.AddEnabledVersion(secRec.ID, []byte("two"))
	got, err := api.AccessSecretVersion(secretprovider.AccessSecretVersionInput{SecretID: secRec.ID, Revision: secretprovider.RevisionLatestEnabled})
	if err != nil {
		t.Fatalf("access latest enabled: %v", err)
	}
	if got.Revision != 2 || string(got.Data) != "two" {
		t.Fatalf("unexpected latest version: %#v", got)
	}
}

func TestCreateSecretAndVersionBranches(t *testing.T) {
	api := NewFakeSecretAPI()
	api.CreateSecretErr = errors.New("create secret boom")
	if _, err := api.CreateSecret(secretprovider.CreateSecretInput{Name: "x-dev"}); err == nil {
		t.Fatal("expected create secret error")
	}
	api.CreateSecretErr = nil

	created, err := api.CreateSecret(secretprovider.CreateSecretInput{Name: "x-dev", Type: secretprovider.SecretTypeOpaque})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}
	if created.Path != "/" {
		t.Fatalf("expected default path '/', got %q", created.Path)
	}
	createdWithPath, err := api.CreateSecret(secretprovider.CreateSecretInput{
		Name: "x2-dev",
		Path: "/custom",
		Type: secretprovider.SecretTypeOpaque,
	})
	if err != nil {
		t.Fatalf("create secret with path: %v", err)
	}
	if createdWithPath.Path != "/custom" {
		t.Fatalf("expected custom path, got %q", createdWithPath.Path)
	}

	api.CreateVerErr = errors.New("create version boom")
	if _, err := api.CreateSecretVersion(secretprovider.CreateSecretVersionInput{SecretID: created.ID, Data: []byte("v1")}); err == nil {
		t.Fatal("expected create version error")
	}
	api.CreateVerErr = nil

	if _, err := api.CreateSecretVersion(secretprovider.CreateSecretVersionInput{SecretID: "missing", Data: []byte("v1")}); err == nil {
		t.Fatal("expected unknown secret error")
	}

	first, err := api.CreateSecretVersion(secretprovider.CreateSecretVersionInput{SecretID: created.ID, Data: []byte("v1")})
	if err != nil {
		t.Fatalf("create first version: %v", err)
	}
	disable := true
	second, err := api.CreateSecretVersion(secretprovider.CreateSecretVersionInput{
		SecretID:        created.ID,
		Data:            []byte("v2"),
		DisablePrevious: &disable,
	})
	if err != nil {
		t.Fatalf("create second version: %v", err)
	}
	if first.Revision != 1 || second.Revision != 2 {
		t.Fatalf("unexpected revisions: first=%d second=%d", first.Revision, second.Revision)
	}

	versions := api.Versions[created.ID]
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].Enabled {
		t.Fatal("expected first version to be disabled")
	}
	if !versions[1].Enabled {
		t.Fatal("expected second version to be enabled")
	}
}
