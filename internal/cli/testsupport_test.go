package cli

import "errors"

type failingWriter struct{}

func (*failingWriter) Write(p []byte) (int, error) { return 0, errors.New("nope") }

type stubSecretAPI struct {
	listFn        func(req ListSecretsInput) ([]SecretRecord, error)
	accessFn      func(req AccessSecretVersionInput) (*SecretVersionRecord, error)
	createSecret  func(req CreateSecretInput) (*SecretRecord, error)
	createVersion func(req CreateSecretVersionInput) (*SecretVersionRecord, error)
}

func (s *stubSecretAPI) ListSecrets(req ListSecretsInput) ([]SecretRecord, error) {
	return s.listFn(req)
}

func (s *stubSecretAPI) AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error) {
	return s.accessFn(req)
}

func (s *stubSecretAPI) CreateSecret(req CreateSecretInput) (*SecretRecord, error) {
	return s.createSecret(req)
}

func (s *stubSecretAPI) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	return s.createVersion(req)
}
