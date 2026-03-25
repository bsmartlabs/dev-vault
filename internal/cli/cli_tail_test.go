package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type createSecretNoPersist struct{ inner *fakeSecretAPI }

func (c *createSecretNoPersist) ListSecrets(req ListSecretsInput) ([]SecretRecord, error) {
	return c.inner.ListSecrets(req)
}
func (c *createSecretNoPersist) AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error) {
	return c.inner.AccessSecretVersion(req)
}
func (c *createSecretNoPersist) CreateSecret(req CreateSecretInput) (*SecretRecord, error) {
	// Do not persist.
	if c.inner.CreateSecretErr != nil {
		return nil, c.inner.CreateSecretErr
	}
	return &SecretRecord{ID: "tmp", ProjectID: "proj", Name: req.Name, Path: "/", Type: req.Type}, nil
}
func (c *createSecretNoPersist) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	return c.inner.CreateSecretVersion(req)
}

func TestPrintUsageCoverage(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"dev-vault", "--help"}, &out, io.Discard, Dependencies{})
	if code != 0 {
		t.Fatalf("expected help exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "dev-vault") {
		t.Fatalf("expected dev-vault in usage")
	}
}

// Ensure failingWriter satisfies io.Writer to silence lints and cover interface use.
var _ io.Writer = (*failingWriter)(nil)
