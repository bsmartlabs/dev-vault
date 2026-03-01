package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
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
	if c.inner.createSecretErr != nil {
		return nil, c.inner.createSecretErr
	}
	return &SecretRecord{ID: "tmp", ProjectID: req.ProjectID, Name: req.Name, Path: "/", Type: req.Type}, nil
}
func (c *createSecretNoPersist) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	return c.inner.CreateSecretVersion(req)
}

func TestPrintUsage_Coverage(t *testing.T) {
	var b bytes.Buffer
	printMainUsage(&b)
	if !strings.Contains(b.String(), config.DefaultConfigName) {
		t.Fatalf("expected config name in usage")
	}
}

// Ensure failingWriter satisfies io.Writer to silence lints and cover interface use.
var _ io.Writer = (*failingWriter)(nil)
