package contracts

import (
	"errors"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type fakeSecretAPI struct {
	listFn func(secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error)
}

func (f fakeSecretAPI) ListSecrets(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	return f.listFn(req)
}

func (f fakeSecretAPI) AccessSecretVersion(secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	return nil, errors.New("not implemented")
}

func (f fakeSecretAPI) CreateSecret(secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (f fakeSecretAPI) CreateSecretVersion(secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	return nil, errors.New("not implemented")
}

func TestRunReadOnlyListSuite(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		called := 0
		seenDefaultPath := false
		api := fakeSecretAPI{listFn: func(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
			called++
			if req.Path == "/" {
				seenDefaultPath = true
			}
			if req.Type == secretprovider.SecretType("not-a-secret-type") {
				return nil, errors.New("invalid secret type \"not-a-secret-type\"")
			}
			return []secretprovider.SecretRecord{}, nil
		}}

		err := RunReadOnlyListSuite(api, "", []secretprovider.SecretType{secretprovider.SecretTypeOpaque, secretprovider.SecretTypeKeyValue})
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if called != 4 {
			t.Fatalf("expected 4 list calls, got %d", called)
		}
		if !seenDefaultPath {
			t.Fatal("expected default path '/' to be used when path is empty")
		}
	})

	t.Run("TypeListFailure", func(t *testing.T) {
		api := fakeSecretAPI{listFn: func(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
			if req.Type == secretprovider.SecretTypeOpaque {
				return nil, errors.New("boom")
			}
			return nil, nil
		}}
		err := RunReadOnlyListSuite(api, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err == nil || !strings.Contains(err.Error(), "list with type") {
			t.Fatalf("expected type failure, got %v", err)
		}
	})

	t.Run("InvalidTypeShapeFailure", func(t *testing.T) {
		api := fakeSecretAPI{listFn: func(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
			if req.Type == secretprovider.SecretType("not-a-secret-type") {
				return nil, errors.New("bad")
			}
			return nil, nil
		}}
		err := RunReadOnlyListSuite(api, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err == nil || !strings.Contains(err.Error(), "error shape mismatch") {
			t.Fatalf("expected invalid type shape failure, got %v", err)
		}
	})

	t.Run("NameFilterFailure", func(t *testing.T) {
		api := fakeSecretAPI{listFn: func(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
			if req.Name != "" {
				return nil, errors.New("name filter failed")
			}
			if req.Type == secretprovider.SecretType("not-a-secret-type") {
				return nil, errors.New("invalid secret type \"not-a-secret-type\"")
			}
			return nil, nil
		}}
		err := RunReadOnlyListSuite(api, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err == nil || !strings.Contains(err.Error(), "list with name filter failed") {
			t.Fatalf("expected name filter failure, got %v", err)
		}
	})

	t.Run("InvalidTypeNoError", func(t *testing.T) {
		api := fakeSecretAPI{listFn: func(secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
			return nil, nil
		}}
		err := RunReadOnlyListSuite(api, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err == nil || !strings.Contains(err.Error(), "invalid type did not fail validation") {
			t.Fatalf("expected invalid type no-error failure, got %v", err)
		}
	})
}

func TestRunReadOnlyListSuiteWithOpen(t *testing.T) {
	api := fakeSecretAPI{listFn: func(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
		if req.Type == secretprovider.SecretType("not-a-secret-type") {
			return nil, errors.New("invalid secret type \"not-a-secret-type\"")
		}
		return nil, nil
	}}

	t.Run("OpenError", func(t *testing.T) {
		err := RunReadOnlyListSuiteWithOpen(func() (secretprovider.SecretAPI, error) {
			return nil, errors.New("boom")
		}, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err == nil || !strings.Contains(err.Error(), "open provider for contracts") {
			t.Fatalf("expected open error, got %v", err)
		}
	})

	t.Run("Success", func(t *testing.T) {
		err := RunReadOnlyListSuiteWithOpen(func() (secretprovider.SecretAPI, error) {
			return api, nil
		}, "/", []secretprovider.SecretType{secretprovider.SecretTypeOpaque})
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
}
