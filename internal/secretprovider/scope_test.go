package secretprovider

import "testing"

type scopeCaptureAPI struct {
	listReq          ListSecretsInput
	accessReq        AccessSecretVersionInput
	createSecretReq  CreateSecretInput
	createVersionReq CreateSecretVersionInput
}

func (s *scopeCaptureAPI) ListSecrets(req ListSecretsInput) ([]SecretRecord, error) {
	s.listReq = req
	return []SecretRecord{{ID: "s1"}}, nil
}

func (s *scopeCaptureAPI) AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error) {
	s.accessReq = req
	return &SecretVersionRecord{SecretID: req.SecretID}, nil
}

func (s *scopeCaptureAPI) CreateSecret(req CreateSecretInput) (*SecretRecord, error) {
	s.createSecretReq = req
	return &SecretRecord{ID: "s2", Name: req.Name, Path: req.Path, Type: req.Type}, nil
}

func (s *scopeCaptureAPI) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	s.createVersionReq = req
	return &SecretVersionRecord{SecretID: req.SecretID, Revision: 1}, nil
}

func TestBindScope_InsertsDefaults(t *testing.T) {
	base := &scopeCaptureAPI{}
	api := BindScope(base, "fr-par", "proj")

	if _, err := api.ListSecrets(ListSecretsInput{Name: "x-dev", Type: SecretTypeOpaque}); err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if base.listReq.Region != "fr-par" || base.listReq.ProjectID != "proj" {
		t.Fatalf("unexpected list scoped req: %#v", base.listReq)
	}

	if _, err := api.AccessSecretVersion(AccessSecretVersionInput{SecretID: "s1", Revision: RevisionLatestEnabled}); err != nil {
		t.Fatalf("AccessSecretVersion: %v", err)
	}
	if base.accessReq.Region != "fr-par" {
		t.Fatalf("unexpected access scoped req: %#v", base.accessReq)
	}

	if _, err := api.CreateSecret(CreateSecretInput{Name: "x-dev", Path: "/", Type: SecretTypeOpaque}); err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if base.createSecretReq.Region != "fr-par" || base.createSecretReq.ProjectID != "proj" {
		t.Fatalf("unexpected create secret scoped req: %#v", base.createSecretReq)
	}

	if _, err := api.CreateSecretVersion(CreateSecretVersionInput{SecretID: "s1", Data: []byte("x")}); err != nil {
		t.Fatalf("CreateSecretVersion: %v", err)
	}
	if base.createVersionReq.Region != "fr-par" {
		t.Fatalf("unexpected create version scoped req: %#v", base.createVersionReq)
	}
}

func TestBindScope_PreservesExplicitValues(t *testing.T) {
	base := &scopeCaptureAPI{}
	api := BindScope(base, "fr-par", "proj")

	_, _ = api.ListSecrets(ListSecretsInput{Region: "nl-ams", ProjectID: "p2", Type: SecretTypeOpaque})
	if base.listReq.Region != "nl-ams" || base.listReq.ProjectID != "p2" {
		t.Fatalf("explicit list scope should be preserved: %#v", base.listReq)
	}

	_, _ = api.CreateSecret(CreateSecretInput{Region: "nl-ams", ProjectID: "p2", Name: "x-dev", Path: "/", Type: SecretTypeOpaque})
	if base.createSecretReq.Region != "nl-ams" || base.createSecretReq.ProjectID != "p2" {
		t.Fatalf("explicit create secret scope should be preserved: %#v", base.createSecretReq)
	}
}
