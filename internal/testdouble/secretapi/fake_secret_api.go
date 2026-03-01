package secretapi

import (
	"errors"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type FakeVersion struct {
	Revision    uint32
	Enabled     bool
	Data        []byte
	Description *string
}

type FakeSecretAPI struct {
	ListErr         error
	AccessErr       error
	CreateSecretErr error
	CreateVerErr    error

	ListCalls int

	Secrets  []secretprovider.SecretRecord
	Versions map[string][]FakeVersion

	secretIDFor func(projectID, name string, existing []secretprovider.SecretRecord) string
}

func NewFakeSecretAPI() *FakeSecretAPI {
	return &FakeSecretAPI{
		Secrets:  []secretprovider.SecretRecord{},
		Versions: make(map[string][]FakeVersion),
		secretIDFor: func(projectID, name string, existing []secretprovider.SecretRecord) string {
			return fmt.Sprintf("sec-%d", len(existing)+1)
		},
	}
}

func NewDeterministicFakeSecretAPI() *FakeSecretAPI {
	api := NewFakeSecretAPI()
	api.secretIDFor = func(projectID, name string, existing []secretprovider.SecretRecord) string {
		return "sec-" + name + "-" + projectID
	}
	return api
}

func (f *FakeSecretAPI) AddSecret(projectID, name, path string, typ secret.SecretType) *secretprovider.SecretRecord {
	id := f.secretIDFor(projectID, name, f.Secrets)
	s := secretprovider.SecretRecord{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Path:      path,
		Type:      secretprovider.SecretType(typ),
	}
	f.Secrets = append(f.Secrets, s)
	return &f.Secrets[len(f.Secrets)-1]
}

func (f *FakeSecretAPI) AddEnabledVersion(secretID string, data []byte) uint32 {
	rev := uint32(len(f.Versions[secretID]) + 1)
	f.Versions[secretID] = append(f.Versions[secretID], FakeVersion{
		Revision: rev,
		Enabled:  true,
		Data:     data,
	})
	return rev
}

func (f *FakeSecretAPI) ListSecrets(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	f.ListCalls++
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	var out []secretprovider.SecretRecord
	for _, s := range f.Secrets {
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

func (f *FakeSecretAPI) AccessSecretVersion(req secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	if f.AccessErr != nil {
		return nil, f.AccessErr
	}
	s := f.findSecret(req.SecretID)
	if s == nil {
		return nil, errors.New("unknown secret")
	}
	versions := f.Versions[req.SecretID]
	var chosen *FakeVersion
	switch req.Revision {
	case secretprovider.RevisionLatestEnabled:
		for i := range versions {
			v := versions[i]
			if v.Enabled {
				if chosen == nil || v.Revision > chosen.Revision {
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
	return &secretprovider.SecretVersionRecord{
		SecretID: req.SecretID,
		Revision: chosen.Revision,
		Data:     chosen.Data,
		Type:     s.Type,
	}, nil
}

func (f *FakeSecretAPI) CreateSecret(req secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	if f.CreateSecretErr != nil {
		return nil, f.CreateSecretErr
	}
	path := "/"
	if req.Path != "" {
		path = req.Path
	}
	return f.AddSecret("proj", req.Name, path, secret.SecretType(req.Type)), nil
}

func (f *FakeSecretAPI) CreateSecretVersion(req secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	if f.CreateVerErr != nil {
		return nil, f.CreateVerErr
	}
	if f.findSecret(req.SecretID) == nil {
		return nil, errors.New("unknown secret")
	}
	rev := uint32(len(f.Versions[req.SecretID]) + 1)
	if req.DisablePrevious != nil && *req.DisablePrevious {
		for i := len(f.Versions[req.SecretID]) - 1; i >= 0; i-- {
			if f.Versions[req.SecretID][i].Enabled {
				f.Versions[req.SecretID][i].Enabled = false
				break
			}
		}
	}
	f.Versions[req.SecretID] = append(f.Versions[req.SecretID], FakeVersion{
		Revision:    rev,
		Enabled:     true,
		Data:        append([]byte(nil), req.Data...),
		Description: req.Description,
	})
	return &secretprovider.SecretVersionRecord{
		Revision: rev,
		SecretID: req.SecretID,
		Status:   "enabled",
	}, nil
}

func (f *FakeSecretAPI) findSecret(id string) *secretprovider.SecretRecord {
	for i := range f.Secrets {
		if f.Secrets[i].ID == id {
			return &f.Secrets[i]
		}
	}
	return nil
}
