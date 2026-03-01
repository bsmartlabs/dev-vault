package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type fakeSecretAPI struct {
	listErr         error
	accessErr       error
	createSecretErr error
	createVerErr    error

	listCalls int

	secrets  []SecretRecord
	versions map[string][]fakeVersion // secretID -> versions (1-based)
}

type fakeVersion struct {
	revision    uint32
	enabled     bool
	data        []byte
	description *string
}

func newFakeSecretAPI() *fakeSecretAPI {
	return &fakeSecretAPI{
		secrets:  []SecretRecord{},
		versions: make(map[string][]fakeVersion),
	}
}

func (f *fakeSecretAPI) AddSecret(projectID, name, path string, typ secret.SecretType) *SecretRecord {
	id := fmt.Sprintf("sec-%d", len(f.secrets)+1)
	s := SecretRecord{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Path:      path,
		Type:      SecretType(typ),
	}
	f.secrets = append(f.secrets, s)
	return &f.secrets[len(f.secrets)-1]
}

func (f *fakeSecretAPI) AddEnabledVersion(secretID string, data []byte) uint32 {
	rev := uint32(len(f.versions[secretID]) + 1)
	f.versions[secretID] = append(f.versions[secretID], fakeVersion{
		revision: rev,
		enabled:  true,
		data:     data,
	})
	return rev
}

func (f *fakeSecretAPI) ListSecrets(req ListSecretsInput) ([]SecretRecord, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []SecretRecord
	for _, s := range f.secrets {
		if req.ProjectID != "" && s.ProjectID != req.ProjectID {
			continue
		}
		if req.Name != "" && s.Name != req.Name {
			continue
		}
		if req.Path != "" && s.Path != req.Path {
			continue
		}
		if req.Type != "" && s.Type != req.Type {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeSecretAPI) AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error) {
	if f.accessErr != nil {
		return nil, f.accessErr
	}
	s := f.findSecret(req.SecretID)
	if s == nil {
		return nil, errors.New("unknown secret")
	}
	versions := f.versions[req.SecretID]
	var chosen *fakeVersion
	switch req.Revision {
	case RevisionLatestEnabled:
		for i := range versions {
			v := versions[i]
			if v.enabled {
				if chosen == nil || v.revision > chosen.revision {
					chosen = &v
				}
			}
		}
	default:
		return nil, errors.New("unsupported revision selector")
	}
	if chosen == nil {
		return nil, errors.New("no enabled version")
	}
	return &SecretVersionRecord{
		SecretID: req.SecretID,
		Revision: chosen.revision,
		Data:     chosen.data,
		Type:     s.Type,
	}, nil
}

func (f *fakeSecretAPI) CreateSecret(req CreateSecretInput) (*SecretRecord, error) {
	if f.createSecretErr != nil {
		return nil, f.createSecretErr
	}
	path := "/"
	if req.Path != "" {
		path = req.Path
	}
	s := f.AddSecret(req.ProjectID, req.Name, path, secret.SecretType(req.Type))
	return s, nil
}

func (f *fakeSecretAPI) CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error) {
	if f.createVerErr != nil {
		return nil, f.createVerErr
	}
	s := f.findSecret(req.SecretID)
	if s == nil {
		return nil, errors.New("unknown secret")
	}
	rev := uint32(len(f.versions[req.SecretID]) + 1)
	if req.DisablePrevious != nil && *req.DisablePrevious {
		// Disable the latest enabled version if any.
		for i := len(f.versions[req.SecretID]) - 1; i >= 0; i-- {
			if f.versions[req.SecretID][i].enabled {
				f.versions[req.SecretID][i].enabled = false
				break
			}
		}
	}
	f.versions[req.SecretID] = append(f.versions[req.SecretID], fakeVersion{
		revision:    rev,
		enabled:     true,
		data:        append([]byte(nil), req.Data...),
		description: req.Description,
	})
	return &SecretVersionRecord{
		Revision: rev,
		SecretID: req.SecretID,
		Status:   "enabled",
	}, nil
}

func (f *fakeSecretAPI) findSecret(id string) *SecretRecord {
	for i := range f.secrets {
		if f.secrets[i].ID == id {
			return &f.secrets[i]
		}
	}
	return nil
}

func writeConfig(t *testing.T, dir string, cfg string) string {
	t.Helper()
	p := filepath.Join(dir, config.DefaultConfigName)
	if err := os.WriteFile(p, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func baseDeps(open func(cfg config.Config, profileOverride string) (SecretAPI, error)) Dependencies {
	return Dependencies{
		Version:       "v",
		Commit:        "c",
		Date:          "d",
		OpenSecretAPI: open,
		Now:           func() time.Time { return time.Unix(123, 0) },
		Hostname:      func() (string, error) { return "host", nil },
		Getwd:         os.Getwd,
	}
}
