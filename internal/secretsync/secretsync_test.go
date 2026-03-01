package secretsync

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type deterministicFakeSecretAPI struct {
	listErr         error
	accessErr       error
	createSecretErr error
	createVerErr    error

	secrets  []secretprovider.SecretRecord
	versions map[string][]deterministicFakeVersion
}

type deterministicFakeVersion struct {
	revision    uint32
	enabled     bool
	data        []byte
	description *string
}

func newDeterministicFakeSecretAPI() *deterministicFakeSecretAPI {
	return &deterministicFakeSecretAPI{
		secrets:  []secretprovider.SecretRecord{},
		versions: make(map[string][]deterministicFakeVersion),
	}
}

func (f *deterministicFakeSecretAPI) AddSecret(projectID, name, path string, typ secret.SecretType) *secretprovider.SecretRecord {
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

func (f *deterministicFakeSecretAPI) AddEnabledVersion(secretID string, data []byte) uint32 {
	rev := uint32(len(f.versions[secretID]) + 1)
	f.versions[secretID] = append(f.versions[secretID], deterministicFakeVersion{
		revision: rev,
		enabled:  true,
		data:     data,
	})
	return rev
}

func (f *deterministicFakeSecretAPI) ListSecrets(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []secretprovider.SecretRecord
	for _, s := range f.secrets {
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

func (f *deterministicFakeSecretAPI) AccessSecretVersion(req secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	if f.accessErr != nil {
		return nil, f.accessErr
	}
	s := f.findSecret(req.SecretID)
	if s == nil {
		return nil, errors.New("unknown secret")
	}
	versions := f.versions[req.SecretID]
	var chosen *deterministicFakeVersion
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

func (f *deterministicFakeSecretAPI) CreateSecret(req secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	if f.createSecretErr != nil {
		return nil, f.createSecretErr
	}
	path := "/"
	if req.Path != "" {
		path = req.Path
	}
	return f.AddSecret("proj", req.Name, path, secret.SecretType(req.Type)), nil
}

func (f *deterministicFakeSecretAPI) CreateSecretVersion(req secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
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
	f.versions[req.SecretID] = append(f.versions[req.SecretID], deterministicFakeVersion{
		revision:    rev,
		enabled:     true,
		data:        append([]byte(nil), req.Data...),
		description: req.Description,
	})
	return &secretprovider.SecretVersionRecord{Revision: rev, SecretID: req.SecretID, Status: "enabled"}, nil
}

func (f *deterministicFakeSecretAPI) findSecret(id string) *secretprovider.SecretRecord {
	for i := range f.secrets {
		if f.secrets[i].ID == id {
			return &f.secrets[i]
		}
	}
	return nil
}

func baseService(root string, _ map[string]mapping.Entry, api secretprovider.SecretAPI) Service {
	svc, err := New(Config{Root: root}, api, Dependencies{
		Now:      func() time.Time { return time.Unix(123, 0) },
		Hostname: func() (string, error) { return "host", nil },
	})
	if err != nil {
		panic(err)
	}
	return svc
}

func TestNew_DefaultsAndInjectedDeps(t *testing.T) {
	api := newDeterministicFakeSecretAPI()
	svc, err := New(Config{Root: "/tmp"}, api, Dependencies{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if svc.api == nil {
		t.Fatalf("expected api to be set")
	}
	if svc.now == nil || svc.hostname == nil || svc.resolvePath == nil {
		t.Fatalf("expected default deps to be set: %#v", svc)
	}
	svcInjected, err := New(Config{Root: "/project"}, api, Dependencies{
		Now:      func() time.Time { return time.Unix(456, 0) },
		Hostname: func() (string, error) { return "x", nil },
	})
	if err != nil {
		t.Fatalf("new injected service: %v", err)
	}
	if svcInjected.cfg.Root != "/project" {
		t.Fatalf("unexpected root: %q", svcInjected.cfg.Root)
	}
	if got := svcInjected.now().Unix(); got != 456 {
		t.Fatalf("unexpected now value: %d", got)
	}
}

func TestNew_RejectsNilAPI(t *testing.T) {
	_, err := New(Config{Root: "/tmp"}, nil, Dependencies{})
	if err == nil {
		t.Fatal("expected error for nil secret api")
	}
	if got := err.Error(); got != "secretsync: nil secret api" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestMappingModeAllows(t *testing.T) {
	if !mapping.ModePull.AllowsPull() {
		t.Fatal("pull mode should allow pull operations")
	}
	if mapping.ModePull.AllowsPush() {
		t.Fatal("pull mode should not allow push operations")
	}
	if mapping.ModePush.AllowsPull() {
		t.Fatal("push mode should not allow pull operations")
	}
	if !mapping.ModePush.AllowsPush() {
		t.Fatal("push mode should allow push operations")
	}
}

func TestBatchOperationErrorMessage(t *testing.T) {
	err := (&BatchOperationError{Operation: "pull", Failed: 1, Total: 3}).Error()
	if err != "pull completed with failures: 1/3 failed" {
		t.Fatalf("unexpected error string: %q", err)
	}
}

func firstPullBatchError(result PullBatchResult) error {
	for _, failure := range result.Failed {
		if failure.Err != nil {
			return failure.Err
		}
	}
	return result.Summary.ErrorOrNil()
}

func firstPushBatchError(result PushBatchResult) error {
	for _, failure := range result.Failed {
		if failure.Err != nil {
			return failure.Err
		}
	}
	return result.Summary.ErrorOrNil()
}

func TestLookupMappedSecret(t *testing.T) {
	api := newDeterministicFakeSecretAPI()
	svc := baseService(t.TempDir(), nil, api)

	api.listErr = errors.New("boom")
	if _, err := svc.lookupMappedSecret("x-dev", mapping.Entry{Path: "/"}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected list error, got %v", err)
	}
	api.listErr = nil

	if _, err := svc.lookupMappedSecret("x-dev", mapping.Entry{Path: "/"}); err == nil {
		t.Fatal("expected not found")
	}

	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	if _, err := svc.lookupMappedSecret("dup-dev", mapping.Entry{Path: "/"}); err == nil || !strings.Contains(err.Error(), "multiple secrets") {
		t.Fatalf("expected multiple match error, got %v", err)
	}

	api = newDeterministicFakeSecretAPI()
	api.AddSecret("proj", "typed-dev", "/", secret.SecretTypeOpaque)
	svc = baseService(t.TempDir(), nil, api)
	got, err := svc.lookupMappedSecret("typed-dev", mapping.Entry{Path: "/", Type: "opaque"})
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
	api := newDeterministicFakeSecretAPI()
	api.listErr = errors.New("boom")
	svc := baseService(t.TempDir(), nil, api)
	if _, err := svc.List(ListQuery{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected list error, got %v", err)
	}

	api = newDeterministicFakeSecretAPI()
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
	api := newDeterministicFakeSecretAPI()
	svc := baseService(root, nil, api)

	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, false); result.Summary.ErrorOrNil() == nil || firstPullBatchError(result) == nil {
		t.Fatal("expected resolve file error")
	}

	if result := svc.PullBatch([]mapping.Target{{Name: "missing-dev", Entry: mapping.Entry{File: "out", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, false); result.Summary.ErrorOrNil() == nil || firstPullBatchError(result) == nil {
		t.Fatal("expected lookup error")
	}

	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.accessErr = errors.New("access boom")
	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "out", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, false); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPullBatchError(result).Error(), "access") {
		t.Fatalf("expected access error, got %v", result.Summary.ErrorOrNil())
	}
	api.accessErr = nil

	api.AddEnabledVersion(sec.ID, []byte("not-json"))
	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "dotenv.env", Path: "/", Format: "dotenv", Mode: mapping.ModePull}}}, true); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPullBatchError(result).Error(), "format dotenv") {
		t.Fatalf("expected dotenv conversion error, got %v", result.Summary.ErrorOrNil())
	}

	api = newDeterministicFakeSecretAPI()
	sec = api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte(`{"A":"1"}`))
	svc = baseService(root, nil, api)
	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "dotenv-success.env", Path: "/", Format: "dotenv", Mode: mapping.ModePull}}}, true); result.Summary.ErrorOrNil() != nil {
		t.Fatalf("expected dotenv conversion success, got %v (%#v)", result.Summary.ErrorOrNil(), result)
	}

	api = newDeterministicFakeSecretAPI()
	sec = api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	svc = baseService(root, nil, api)

	existingPath := filepath.Join(root, "exists.txt")
	if err := os.WriteFile(existingPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "exists.txt", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, false); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPullBatchError(result).Error(), "file exists") {
		t.Fatalf("expected exists error, got %v", result.Summary.ErrorOrNil())
	}

	notDir := filepath.Join(root, "notdir")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	if result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "notdir/out.txt", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, true); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPullBatchError(result).Error(), "write") {
		t.Fatalf("expected generic write error, got %v", result.Summary.ErrorOrNil())
	}

	result := svc.PullBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "ok.bin", Path: "/", Format: "raw", Mode: mapping.ModePull}}}, true)
	if err := result.Summary.ErrorOrNil(); err != nil {
		t.Fatalf("unexpected pull error: %v", err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Name != "x-dev" {
		t.Fatalf("unexpected pull results: %#v", result)
	}
}

func TestPullBatch_ReturnsPerTargetOutcomes(t *testing.T) {
	root := t.TempDir()
	api := newDeterministicFakeSecretAPI()
	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	svc := baseService(root, nil, api)

	result := svc.PullBatch([]mapping.Target{
		{Name: "x-dev", Entry: mapping.Entry{File: "ok.bin", Path: "/", Format: mapping.FormatRaw, Mode: mapping.ModePull}},
		{Name: "missing-dev", Entry: mapping.Entry{File: "missing.bin", Path: "/", Format: mapping.FormatRaw, Mode: mapping.ModePull}},
	}, true)
	err := result.Summary.ErrorOrNil()
	if err == nil {
		t.Fatal("expected batch error")
	}
	var batchErr *BatchOperationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected BatchOperationError, got %T", err)
	}
	if batchErr.Operation != "pull" || batchErr.Failed != 1 || batchErr.Total != 2 {
		t.Fatalf("unexpected batch error: %#v", batchErr)
	}
	if len(result.Succeeded) != 1 {
		t.Fatalf("unexpected successes length: %d", len(result.Succeeded))
	}
	if result.Succeeded[0].Name != "x-dev" {
		t.Fatalf("unexpected first outcome: %#v", result.Succeeded[0])
	}
	if len(result.Failed) != 1 || result.Failed[0].Err == nil || result.Failed[0].Name != "missing-dev" {
		t.Fatalf("unexpected failed outcomes: %#v", result.Failed)
	}
}

func TestPullBatch_AllSuccess(t *testing.T) {
	root := t.TempDir()
	api := newDeterministicFakeSecretAPI()
	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	svc := baseService(root, nil, api)

	result := svc.PullBatch([]mapping.Target{
		{Name: "x-dev", Entry: mapping.Entry{File: "ok.bin", Path: "/", Format: mapping.FormatRaw, Mode: mapping.ModePull}},
	}, true)
	if err := result.Summary.ErrorOrNil(); err != nil {
		t.Fatalf("unexpected pull batch error: %v", err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Revision == 0 {
		t.Fatalf("unexpected pull batch results: %#v", result)
	}
}

func TestPushHelpersAndPush(t *testing.T) {
	root := t.TempDir()
	api := newDeterministicFakeSecretAPI()
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

	if _, err := svc.readPushPayload("x-dev", mapping.Entry{File: "", Format: "raw"}); err == nil {
		t.Fatal("expected resolve file error")
	}
	if _, err := svc.readPushPayload("x-dev", mapping.Entry{File: "missing.bin", Format: "raw"}); err == nil {
		t.Fatal("expected read file error")
	}

	if err := os.WriteFile(filepath.Join(root, "bad.env"), []byte("BAD"), 0o600); err != nil {
		t.Fatalf("write bad env: %v", err)
	}
	if _, err := svc.readPushPayload("x-dev", mapping.Entry{File: "bad.env", Format: "dotenv"}); err == nil {
		t.Fatal("expected dotenv parse error")
	}

	if err := os.WriteFile(filepath.Join(root, "ok.env"), []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("write ok env: %v", err)
	}
	if _, err := svc.readPushPayload("x-dev", mapping.Entry{File: "ok.env", Format: "dotenv"}); err != nil {
		t.Fatalf("unexpected dotenv conversion error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "raw.bin"), []byte("RAW"), 0o600); err != nil {
		t.Fatalf("write raw file: %v", err)
	}
	if payload, err := svc.readPushPayload("x-dev", mapping.Entry{File: "raw.bin", Format: "raw"}); err != nil || string(payload) != "RAW" {
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

	if _, err := svc.lookupMappedSecret("missing-dev", mapping.Entry{Path: "/"}); err == nil {
		t.Fatal("expected lookup error when missing")
	}
	if _, err := svc.lookupOrCreateMappedSecret("missing-dev", mapping.Entry{Path: "/"}); err == nil || !strings.Contains(err.Error(), "create-missing requires mapping.type") {
		t.Fatalf("expected missing type error, got %v", err)
	}

	api.listErr = errors.New("boom")
	if _, err := svc.lookupOrCreateMappedSecret("x-dev", mapping.Entry{Path: "/", Type: "opaque"}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected list error passthrough, got %v", err)
	}
	api.listErr = nil

	api.createSecretErr = errors.New("create secret boom")
	if _, err := svc.lookupOrCreateMappedSecret("x-dev", mapping.Entry{Path: "/", Type: "opaque"}); err == nil || !strings.Contains(err.Error(), "create secret") {
		t.Fatalf("expected create secret error, got %v", err)
	}
	api.createSecretErr = nil

	created, err := svc.lookupOrCreateMappedSecret("x-dev", mapping.Entry{Path: "/", Type: "opaque"})
	if err != nil {
		t.Fatalf("unexpected create missing success error: %v", err)
	}
	if created.Name != "x-dev" {
		t.Fatalf("unexpected created secret: %#v", created)
	}
	existing, err := svc.lookupOrCreateMappedSecret("x-dev", mapping.Entry{Path: "/", Type: "opaque"})
	if err != nil {
		t.Fatalf("unexpected existing lookup/create error: %v", err)
	}
	if existing.Name != "x-dev" {
		t.Fatalf("unexpected existing secret: %#v", existing)
	}
	if _, err := svc.resolveExistingSecretForPush("x-dev", mapping.Entry{Path: "/", Type: "opaque"}); err != nil {
		t.Fatalf("resolveExistingSecretForPush: %v", err)
	}
	if _, err := svc.resolveOrCreateSecretForPush("y-dev", mapping.Entry{Path: "/", Type: "opaque"}); err != nil {
		t.Fatalf("resolveOrCreateSecretForPush: %v", err)
	}
	if _, err := svc.resolveOrCreateSecretForPush("z-dev", mapping.Entry{Path: "/"}); err == nil {
		t.Fatal("expected resolveOrCreateSecretForPush create-missing validation error")
	}

	if err := os.WriteFile(filepath.Join(root, "push.bin"), []byte("PUSH"), 0o600); err != nil {
		t.Fatalf("write push.bin: %v", err)
	}
	if result := svc.PushBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "missing.bin", Path: "/", Type: "opaque", Format: "raw", Mode: mapping.ModePush}}}, PushOptions{}); result.Summary.ErrorOrNil() == nil || firstPushBatchError(result) == nil {
		t.Fatal("expected push read payload error")
	}
	if result := svc.PushBatch([]mapping.Target{{Name: "never-created-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw", Mode: mapping.ModePush}}}, PushOptions{}); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPushBatchError(result).Error(), "resolve never-created-dev") {
		t.Fatalf("expected push resolve error, got %v", result.Summary.ErrorOrNil())
	}
	api.createVerErr = errors.New("version boom")
	if result := svc.PushBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw", Mode: mapping.ModePush}}}, PushOptions{}); result.Summary.ErrorOrNil() == nil || !strings.Contains(firstPushBatchError(result).Error(), "create version") {
		t.Fatalf("expected create version error, got %v", result.Summary.ErrorOrNil())
	}
	api.createVerErr = nil

	result := svc.PushBatch([]mapping.Target{{Name: "x-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw", Mode: mapping.ModePush}}}, PushOptions{DisablePrevious: true})
	if err := result.Summary.ErrorOrNil(); err != nil {
		t.Fatalf("unexpected push success error: %v", err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Name != "x-dev" {
		t.Fatalf("unexpected push results: %#v", result)
	}

	createMissingResult := svc.PushBatch(
		[]mapping.Target{{Name: "new-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: "raw", Mode: mapping.ModePush}}},
		PushOptions{CreateMissing: true},
	)
	if err := createMissingResult.Summary.ErrorOrNil(); err != nil {
		t.Fatalf("unexpected create-missing push error: %v", err)
	}
	if len(createMissingResult.Succeeded) != 1 || createMissingResult.Succeeded[0].Name != "new-dev" {
		t.Fatalf("unexpected create-missing push results: %#v", createMissingResult)
	}
}

func TestPushBatch_ReturnsPerTargetOutcomes(t *testing.T) {
	root := t.TempDir()
	api := newDeterministicFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	svc := baseService(root, nil, api)

	if err := os.WriteFile(filepath.Join(root, "push.bin"), []byte("PUSH"), 0o600); err != nil {
		t.Fatalf("write push.bin: %v", err)
	}

	result := svc.PushBatch([]mapping.Target{
		{Name: "x-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: mapping.FormatRaw, Mode: mapping.ModePush}},
		{Name: "missing-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: mapping.FormatRaw, Mode: mapping.ModePush}},
	}, PushOptions{})
	err := result.Summary.ErrorOrNil()
	if err == nil {
		t.Fatal("expected batch error")
	}
	var batchErr *BatchOperationError
	if !errors.As(err, &batchErr) {
		t.Fatalf("expected BatchOperationError, got %T", err)
	}
	if batchErr.Operation != "push" || batchErr.Failed != 1 || batchErr.Total != 2 {
		t.Fatalf("unexpected batch error: %#v", batchErr)
	}
	if len(result.Succeeded) != 1 {
		t.Fatalf("unexpected successes length: %d", len(result.Succeeded))
	}
	if result.Succeeded[0].Name != "x-dev" || result.Succeeded[0].Revision == 0 {
		t.Fatalf("unexpected first outcome: %#v", result.Succeeded[0])
	}
	if len(result.Failed) != 1 || result.Failed[0].Err == nil || result.Failed[0].Name != "missing-dev" {
		t.Fatalf("unexpected failed outcomes: %#v", result.Failed)
	}
}

func TestPushBatch_AllSuccess(t *testing.T) {
	root := t.TempDir()
	api := newDeterministicFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	svc := baseService(root, nil, api)

	if err := os.WriteFile(filepath.Join(root, "push.bin"), []byte("PUSH"), 0o600); err != nil {
		t.Fatalf("write push.bin: %v", err)
	}

	result := svc.PushBatch([]mapping.Target{
		{Name: "x-dev", Entry: mapping.Entry{File: "push.bin", Path: "/", Type: "opaque", Format: mapping.FormatRaw, Mode: mapping.ModePush}},
	}, PushOptions{})
	if err := result.Summary.ErrorOrNil(); err != nil {
		t.Fatalf("unexpected push batch error: %v", err)
	}
	if len(result.Succeeded) != 1 || result.Succeeded[0].Revision == 0 {
		t.Fatalf("unexpected push batch results: %#v", result)
	}
}
