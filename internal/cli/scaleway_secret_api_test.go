package cli

import (
	"errors"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type fakeScalewaySDK struct {
	listFn          func(*secret.ListSecretsRequest, ...scw.RequestOption) (*secret.ListSecretsResponse, error)
	accessFn        func(*secret.AccessSecretVersionRequest, ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error)
	createSecretFn  func(*secret.CreateSecretRequest, ...scw.RequestOption) (*secret.Secret, error)
	createVersionFn func(*secret.CreateSecretVersionRequest, ...scw.RequestOption) (*secret.SecretVersion, error)
}

func (f *fakeScalewaySDK) ListSecrets(req *secret.ListSecretsRequest, opts ...scw.RequestOption) (*secret.ListSecretsResponse, error) {
	return f.listFn(req, opts...)
}

func (f *fakeScalewaySDK) AccessSecretVersion(req *secret.AccessSecretVersionRequest, opts ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error) {
	return f.accessFn(req, opts...)
}

func (f *fakeScalewaySDK) CreateSecret(req *secret.CreateSecretRequest, opts ...scw.RequestOption) (*secret.Secret, error) {
	return f.createSecretFn(req, opts...)
}

func (f *fakeScalewaySDK) CreateSecretVersion(req *secret.CreateSecretVersionRequest, opts ...scw.RequestOption) (*secret.SecretVersion, error) {
	return f.createVersionFn(req, opts...)
}

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

func TestScalewaySecretAPI_ListSecrets(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.ListSecrets(ListSecretsInput{Region: "bad", ProjectID: "p", Type: "opaque"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.ListSecrets(ListSecretsInput{Region: "fr-par", ProjectID: "p", Type: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("APIError", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			listFn: func(*secret.ListSecretsRequest, ...scw.RequestOption) (*secret.ListSecretsResponse, error) {
				return nil, errors.New("boom")
			},
		}}
		_, err := api.ListSecrets(ListSecretsInput{Region: "fr-par", ProjectID: "p", Type: "opaque"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Success", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			listFn: func(req *secret.ListSecretsRequest, _ ...scw.RequestOption) (*secret.ListSecretsResponse, error) {
				if req.ProjectID == nil || *req.ProjectID != "p" {
					t.Fatalf("unexpected project id: %#v", req.ProjectID)
				}
				if req.Name == nil || *req.Name != "name-dev" {
					t.Fatalf("unexpected name filter: %#v", req.Name)
				}
				if req.Path == nil || *req.Path != "/" {
					t.Fatalf("unexpected path filter: %#v", req.Path)
				}
				return &secret.ListSecretsResponse{Secrets: []*secret.Secret{
					nil,
					{ID: "s1", Name: "name-dev", Path: "/", ProjectID: "p", Type: secret.SecretTypeOpaque},
				}}, nil
			},
		}}
		out, err := api.ListSecrets(ListSecretsInput{
			Region:    "fr-par",
			ProjectID: "p",
			Name:      "name-dev",
			Path:      "/",
			Type:      "opaque",
		})
		if err != nil {
			t.Fatalf("ListSecrets: %v", err)
		}
		if len(out) != 2 || out[1] == nil || out[1].Type != "opaque" {
			t.Fatalf("unexpected output: %#v", out)
		}
	})
}

func TestScalewaySecretAPI_AccessSecretVersion(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.AccessSecretVersion(AccessSecretVersionInput{Region: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("APIError", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			accessFn: func(*secret.AccessSecretVersionRequest, ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error) {
				return nil, errors.New("boom")
			},
		}}
		_, err := api.AccessSecretVersion(AccessSecretVersionInput{Region: "fr-par", SecretID: "s1", Revision: "latest_enabled"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Success", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			accessFn: func(req *secret.AccessSecretVersionRequest, _ ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error) {
				if req.Revision != "latest_enabled" {
					t.Fatalf("unexpected revision: %s", req.Revision)
				}
				return &secret.AccessSecretVersionResponse{
					SecretID: "s1",
					Revision: 3,
					Data:     []byte("x"),
					Type:     secret.SecretTypeOpaque,
				}, nil
			},
		}}
		out, err := api.AccessSecretVersion(AccessSecretVersionInput{Region: "fr-par", SecretID: "s1", Revision: "latest_enabled"})
		if err != nil {
			t.Fatalf("AccessSecretVersion: %v", err)
		}
		if out.Revision != 3 || out.Type != "opaque" {
			t.Fatalf("unexpected output: %#v", out)
		}
	})
}

func TestScalewaySecretAPI_CreateSecret(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.CreateSecret(CreateSecretInput{Region: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.CreateSecret(CreateSecretInput{Region: "fr-par", Type: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("APIError", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			createSecretFn: func(*secret.CreateSecretRequest, ...scw.RequestOption) (*secret.Secret, error) {
				return nil, errors.New("boom")
			},
		}}
		_, err := api.CreateSecret(CreateSecretInput{Region: "fr-par", Type: "opaque"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Success", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			createSecretFn: func(req *secret.CreateSecretRequest, _ ...scw.RequestOption) (*secret.Secret, error) {
				if req.Path == nil || *req.Path != "/" {
					t.Fatalf("expected default path '/'")
				}
				return &secret.Secret{
					ID:        "s1",
					ProjectID: req.ProjectID,
					Name:      req.Name,
					Path:      *req.Path,
					Type:      req.Type,
				}, nil
			},
		}}
		out, err := api.CreateSecret(CreateSecretInput{
			Region:    "fr-par",
			ProjectID: "p",
			Name:      "x-dev",
			Type:      "opaque",
		})
		if err != nil {
			t.Fatalf("CreateSecret: %v", err)
		}
		if out.ID != "s1" || out.Type != "opaque" {
			t.Fatalf("unexpected output: %#v", out)
		}
	})
}

func TestScalewaySecretAPI_CreateSecretVersion(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{}}
		_, err := api.CreateSecretVersion(CreateSecretVersionInput{Region: "bad"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("APIError", func(t *testing.T) {
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			createVersionFn: func(*secret.CreateSecretVersionRequest, ...scw.RequestOption) (*secret.SecretVersion, error) {
				return nil, errors.New("boom")
			},
		}}
		_, err := api.CreateSecretVersion(CreateSecretVersionInput{Region: "fr-par"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Success", func(t *testing.T) {
		desc := "d"
		disable := true
		api := &scwSecretAPI{api: &fakeScalewaySDK{
			createVersionFn: func(req *secret.CreateSecretVersionRequest, _ ...scw.RequestOption) (*secret.SecretVersion, error) {
				if req.Description == nil || *req.Description != desc {
					t.Fatalf("unexpected description: %#v", req.Description)
				}
				if req.DisablePrevious == nil || *req.DisablePrevious != disable {
					t.Fatalf("unexpected disable_previous: %#v", req.DisablePrevious)
				}
				return &secret.SecretVersion{
					SecretID: req.SecretID,
					Revision: 9,
					Status:   secret.SecretVersionStatusEnabled,
				}, nil
			},
		}}
		out, err := api.CreateSecretVersion(CreateSecretVersionInput{
			Region:          "fr-par",
			SecretID:        "s1",
			Data:            []byte("x"),
			Description:     &desc,
			DisablePrevious: &disable,
		})
		if err != nil {
			t.Fatalf("CreateSecretVersion: %v", err)
		}
		if out.Revision != 9 || out.Status != "enabled" {
			t.Fatalf("unexpected output: %#v", out)
		}
	})
}

func TestToScalewaySecretType(t *testing.T) {
	if _, err := toScalewaySecretType("opaque"); err != nil {
		t.Fatalf("opaque should be supported: %v", err)
	}
	if _, err := toScalewaySecretType("not-valid"); err == nil {
		t.Fatal("expected unsupported mapping error")
	}
}
