package secretsync

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type fakeSecretAPI struct {
	listErr         error
	accessErr       error
	createSecretErr error
	createVerErr    error

	secrets  []secretprovider.SecretRecord
	versions map[string][]fakeVersion
}

type fakeVersion struct {
	revision    uint32
	enabled     bool
	data        []byte
	description *string
}

func newFakeSecretAPI() *fakeSecretAPI {
	return &fakeSecretAPI{
		secrets:  []secretprovider.SecretRecord{},
		versions: make(map[string][]fakeVersion),
	}
}

func (f *fakeSecretAPI) AddSecret(projectID, name, path string, typ secret.SecretType) *secretprovider.SecretRecord {
	id := "sec-" + name + "-" + projectID
	s := secretprovider.SecretRecord{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Path:      path,
		Type:      secretprovider.SecretType(typ),
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

func (f *fakeSecretAPI) ListSecrets(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []secretprovider.SecretRecord
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

func (f *fakeSecretAPI) AccessSecretVersion(req secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
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
	case secretprovider.RevisionLatestEnabled:
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
	return &secretprovider.SecretVersionRecord{
		SecretID: req.SecretID,
		Revision: chosen.revision,
		Data:     chosen.data,
		Type:     s.Type,
	}, nil
}

func (f *fakeSecretAPI) CreateSecret(req secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	if f.createSecretErr != nil {
		return nil, f.createSecretErr
	}
	path := "/"
	if req.Path != "" {
		path = req.Path
	}
	return f.AddSecret(req.ProjectID, req.Name, path, secret.SecretType(req.Type)), nil
}

func (f *fakeSecretAPI) CreateSecretVersion(req secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	if f.createVerErr != nil {
		return nil, f.createVerErr
	}
	s := f.findSecret(req.SecretID)
	if s == nil {
		return nil, errors.New("unknown secret")
	}
	rev := uint32(len(f.versions[req.SecretID]) + 1)
	if req.DisablePrevious != nil && *req.DisablePrevious {
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
	return &secretprovider.SecretVersionRecord{Revision: rev, SecretID: req.SecretID, Status: "enabled"}, nil
}

func (f *fakeSecretAPI) findSecret(id string) *secretprovider.SecretRecord {
	for i := range f.secrets {
		if f.secrets[i].ID == id {
			return &f.secrets[i]
		}
	}
	return nil
}

func baseService(root string, mapping map[string]config.MappingEntry, api secretprovider.SecretAPI) Service {
	return New(Config{Root: root, Mapping: mapping}, api, Dependencies{
		Now:      func() time.Time { return time.Unix(123, 0) },
		Hostname: func() (string, error) { return "host", nil },
	})
}

func TestNewAndNewFromLoaded(t *testing.T) {
	api := newFakeSecretAPI()
	svc := New(Config{Root: "/tmp", Mapping: map[string]config.MappingEntry{}}, api, Dependencies{})
	if svc.api == nil {
		t.Fatalf("expected api to be set")
	}
	if svc.now == nil || svc.hostname == nil {
		t.Fatalf("expected default deps to be set")
	}

	loaded := &config.Loaded{Root: "/project", Cfg: config.Config{Mapping: map[string]config.MappingEntry{"a-dev": {Mode: "both"}}}}
	svcFromLoaded := NewFromLoaded(loaded, api, Dependencies{
		Now:      func() time.Time { return time.Unix(456, 0) },
		Hostname: func() (string, error) { return "x", nil },
	})
	if svcFromLoaded.cfg.Root != "/project" {
		t.Fatalf("unexpected root: %q", svcFromLoaded.cfg.Root)
	}
	if got := svcFromLoaded.now().Unix(); got != 456 {
		t.Fatalf("unexpected now value: %d", got)
	}
}

func TestParseType(t *testing.T) {
	if _, err := ParseSecretType("opaque"); err != nil {
		t.Fatalf("expected valid secret type, got %v", err)
	}
	if _, err := ParseSecretType("invalid"); err == nil {
		t.Fatal("expected invalid secret type error")
	}
}

func TestLookupMappedSecret(t *testing.T) {
	api := newFakeSecretAPI()
	svc := baseService(t.TempDir(), nil, api)

	api.listErr = errors.New("boom")
	if _, err := svc.LookupMappedSecret("x-dev", config.MappingEntry{Path: "/"}); err == nil || !strings.Contains(err.Error(), "list secrets") {
		t.Fatalf("expected list error, got %v", err)
	}
	api.listErr = nil

	if _, err := svc.LookupMappedSecret("x-dev", config.MappingEntry{Path: "/"}); err == nil {
		t.Fatal("expected not found")
	}

	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	if _, err := svc.LookupMappedSecret("dup-dev", config.MappingEntry{Path: "/"}); err == nil || !strings.Contains(err.Error(), "multiple secrets") {
		t.Fatalf("expected multiple match error, got %v", err)
	}

	api = newFakeSecretAPI()
	api.AddSecret("proj", "typed-dev", "/", secret.SecretTypeOpaque)
	svc = baseService(t.TempDir(), nil, api)
	got, err := svc.LookupMappedSecret("typed-dev", config.MappingEntry{Path: "/", Type: "opaque"})
	if err != nil {
		t.Fatalf("unexpected lookup error: %v", err)
	}
	if got.Name != "typed-dev" {
		t.Fatalf("unexpected name: %s", got.Name)
	}

	miss := &SecretLookupMissError{Name: "missing-dev", Path: "/"}
	if !strings.Contains(miss.Error(), "missing-dev") {
		t.Fatalf("unexpected error message: %s", miss.Error())
	}
}

func TestList(t *testing.T) {
	api := newFakeSecretAPI()
	api.listErr = errors.New("boom")
	svc := baseService(t.TempDir(), nil, api)
	if _, err := svc.List(ListQuery{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected list error, got %v", err)
	}

	api = newFakeSecretAPI()
	api.AddSecret("proj", "zzz-dev", "/a", secret.SecretTypeOpaque)
	api.AddSecret("proj", "aaa-dev", "/a", secret.SecretTypeKeyValue)
	api.AddSecret("proj", "plain-prod", "/a", secret.SecretTypeOpaque)
	svc = baseService(t.TempDir(), nil, api)

	re, err := regexp.Compile(`^a.*-dev$`)
	if err != nil {
		t.Fatalf("compile regex: %v", err)
	}

	records, err := svc.List(ListQuery{
		NameContains: []string{"a"},
		NameRegex:    re,
		Path:         "/a",
		Type:         secretprovider.SecretTypeKeyValue,
	})
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(records) != 1 || records[0].Name != "aaa-dev" {
		t.Fatalf("unexpected records: %#v", records)
	}

	missFiltered, err := svc.List(ListQuery{NameContains: []string{"nope"}})
	if err != nil {
		t.Fatalf("list with contains miss error: %v", err)
	}
	if len(missFiltered) != 0 {
		t.Fatalf("expected contains miss to filter out all, got %#v", missFiltered)
	}

	regexFiltered, err := svc.List(ListQuery{NameRegex: regexp.MustCompile(`^zzz.*-dev$`)})
	if err != nil {
		t.Fatalf("list with regex filter error: %v", err)
	}
	if len(regexFiltered) != 1 || regexFiltered[0].Name != "zzz-dev" {
		t.Fatalf("unexpected regex-filtered records: %#v", regexFiltered)
	}

	allRecords, err := svc.List(ListQuery{})
	if err != nil {
		t.Fatalf("list all error: %v", err)
	}
	if len(allRecords) != 2 || allRecords[0].Name != "aaa-dev" || allRecords[1].Name != "zzz-dev" {
		t.Fatalf("unexpected sorted records: %#v", allRecords)
	}
}

func TestPull(t *testing.T) {
	root := t.TempDir()
	api := newFakeSecretAPI()
	svc := baseService(root, nil, api)

	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "", Path: "/", Format: "raw"}}}, false); err == nil {
		t.Fatal("expected resolve file error")
	}

	if _, err := svc.Pull([]MappingTarget{{Name: "missing-dev", Entry: config.MappingEntry{File: "out", Path: "/", Format: "raw"}}}, false); err == nil {
		t.Fatal("expected lookup error")
	}

	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.accessErr = errors.New("access boom")
	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "out", Path: "/", Format: "raw"}}}, false); err == nil || !strings.Contains(err.Error(), "access") {
		t.Fatalf("expected access error, got %v", err)
	}
	api.accessErr = nil

	api.AddEnabledVersion(sec.ID, []byte("not-json"))
	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "dotenv.env", Path: "/", Format: "dotenv"}}}, true); err == nil || !strings.Contains(err.Error(), "format dotenv") {
		t.Fatalf("expected dotenv conversion error, got %v", err)
	}

	api = newFakeSecretAPI()
	sec = api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte(`{"A":"1"}`))
	svc = baseService(root, nil, api)
	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "dotenv-success.env", Path: "/", Format: "dotenv"}}}, true); err != nil {
		t.Fatalf("expected dotenv conversion success, got %v", err)
	}

	api = newFakeSecretAPI()
	sec = api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	svc = baseService(root, nil, api)

	existingPath := filepath.Join(root, "exists.txt")
	if err := os.WriteFile(existingPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "exists.txt", Path: "/", Format: "raw"}}}, false); err == nil || !strings.Contains(err.Error(), "file exists") {
		t.Fatalf("expected exists error, got %v", err)
	}

	notDir := filepath.Join(root, "notdir")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	if _, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "notdir/out.txt", Path: "/", Format: "raw"}}}, true); err == nil || !strings.Contains(err.Error(), "write") {
		t.Fatalf("expected generic write error, got %v", err)
	}

	results, err := svc.Pull([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "ok.bin", Path: "/", Format: "raw"}}}, true)
	if err != nil {
		t.Fatalf("unexpected pull error: %v", err)
	}
	if len(results) != 1 || results[0].Name != "x-dev" {
		t.Fatalf("unexpected pull results: %#v", results)
	}
}

func TestPushHelpersAndPush(t *testing.T) {
	root := t.TempDir()
	api := newFakeSecretAPI()
	svc := baseService(root, nil, api)

	if got := svc.pushDescription("explicit"); got != "explicit" {
		t.Fatalf("unexpected explicit description: %q", got)
	}
	if got := svc.pushDescription(""); !strings.Contains(got, "host") {
		t.Fatalf("expected hostname-backed default description, got %q", got)
	}
	svc.hostname = func() (string, error) { return "", errors.New("no host") }
	if got := svc.pushDescription(""); !strings.Contains(got, "unknown-host") {
		t.Fatalf("unexpected default description: %q", got)
	}

	if _, err := svc.readPushPayload("x-dev", config.MappingEntry{File: "", Format: "raw"}); err == nil {
		t.Fatal("expected resolve file error")
	}
	if _, err := svc.readPushPayload("x-dev", config.MappingEntry{File: "missing.bin", Format: "raw"}); err == nil {
		t.Fatal("expected read file error")
	}

	if err := os.WriteFile(filepath.Join(root, "bad.env"), []byte("BAD"), 0o600); err != nil {
		t.Fatalf("write bad env: %v", err)
	}
	if _, err := svc.readPushPayload("x-dev", config.MappingEntry{File: "bad.env", Format: "dotenv"}); err == nil {
		t.Fatal("expected dotenv parse error")
	}

	if err := os.WriteFile(filepath.Join(root, "ok.env"), []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("write ok env: %v", err)
	}
	if _, err := svc.readPushPayload("x-dev", config.MappingEntry{File: "ok.env", Format: "dotenv"}); err != nil {
		t.Fatalf("unexpected dotenv conversion error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "raw.bin"), []byte("RAW"), 0o600); err != nil {
		t.Fatalf("write raw file: %v", err)
	}
	if payload, err := svc.readPushPayload("x-dev", config.MappingEntry{File: "raw.bin", Format: "raw"}); err != nil || string(payload) != "RAW" {
		t.Fatalf("unexpected raw payload: %q err=%v", payload, err)
	}

	req := createSecretVersionInput("sec", []byte("X"), "desc", false)
	if req.DisablePrevious != nil {
		t.Fatalf("expected nil DisablePrevious when false")
	}
	req = createSecretVersionInput("sec", []byte("X"), "desc", true)
	if req.DisablePrevious == nil || !*req.DisablePrevious {
		t.Fatalf("expected DisablePrevious=true")
	}

	if _, err := svc.ResolveMappedSecret("missing-dev", config.MappingEntry{Path: "/"}, false); err == nil {
		t.Fatal("expected resolve error when missing and createMissing=false")
	}
	if _, err := svc.ResolveMappedSecret("missing-dev", config.MappingEntry{Path: "/"}, true); err == nil || !strings.Contains(err.Error(), "create-missing requires mapping.type") {
		t.Fatalf("expected missing type error, got %v", err)
	}

	api.listErr = errors.New("boom")
	if _, err := svc.ResolveMappedSecret("x-dev", config.MappingEntry{Path: "/", Type: "opaque"}, true); err == nil || !strings.Contains(err.Error(), "list secrets") {
		t.Fatalf("expected list error passthrough, got %v", err)
	}
	api.listErr = nil

	api.createSecretErr = errors.New("create secret boom")
	if _, err := svc.ResolveMappedSecret("x-dev", config.MappingEntry{Path: "/", Type: "opaque"}, true); err == nil || !strings.Contains(err.Error(), "create secret") {
		t.Fatalf("expected create secret error, got %v", err)
	}
	api.createSecretErr = nil

	created, err := svc.ResolveMappedSecret("x-dev", config.MappingEntry{Path: "/", Type: "opaque"}, true)
	if err != nil {
		t.Fatalf("unexpected create missing success error: %v", err)
	}
	if created.Name != "x-dev" {
		t.Fatalf("unexpected created secret: %#v", created)
	}

	if err := os.WriteFile(filepath.Join(root, "push.bin"), []byte("PUSH"), 0o600); err != nil {
		t.Fatalf("write push.bin: %v", err)
	}
	if _, err := svc.Push([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "missing.bin", Path: "/", Type: "opaque", Format: "raw"}}}, PushOptions{}); err == nil {
		t.Fatal("expected push read payload error")
	}
	if _, err := svc.Push([]MappingTarget{{Name: "never-created-dev", Entry: config.MappingEntry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw"}}}, PushOptions{}); err == nil || !strings.Contains(err.Error(), "resolve never-created-dev") {
		t.Fatalf("expected push resolve error, got %v", err)
	}
	api.createVerErr = errors.New("version boom")
	if _, err := svc.Push([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw"}}}, PushOptions{}); err == nil || !strings.Contains(err.Error(), "create version") {
		t.Fatalf("expected create version error, got %v", err)
	}
	api.createVerErr = nil

	results, err := svc.Push([]MappingTarget{{Name: "x-dev", Entry: config.MappingEntry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw"}}}, PushOptions{DisablePrevious: true})
	if err != nil {
		t.Fatalf("unexpected push success error: %v", err)
	}
	if len(results) != 1 || results[0].Name != "x-dev" {
		t.Fatalf("unexpected push results: %#v", results)
	}
}
