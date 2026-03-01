package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

	secrets  []*SecretRecord
	versions map[string][]fakeVersion // secretID -> versions (1-based)
}

type fakeVersion struct {
	revision     uint32
	enabled      bool
	data         []byte
	description  *string
	disablePrev  bool
	createdAtUTC time.Time
}

func newFakeSecretAPI() *fakeSecretAPI {
	return &fakeSecretAPI{
		secrets:  []*SecretRecord{},
		versions: make(map[string][]fakeVersion),
	}
}

func (f *fakeSecretAPI) AddSecret(projectID, name, path string, typ secret.SecretType) *SecretRecord {
	id := fmt.Sprintf("sec-%d", len(f.secrets)+1)
	s := &SecretRecord{
		ID:        id,
		ProjectID: projectID,
		Name:      name,
		Path:      path,
		Type:      string(typ),
	}
	f.secrets = append(f.secrets, s)
	return s
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

func (f *fakeSecretAPI) ListSecrets(req ListSecretsInput) ([]*SecretRecord, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	var out []*SecretRecord
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
	case "latest_enabled":
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
	for _, s := range f.secrets {
		if s.ID == id {
			return s
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

func TestRun_GlobalFlagParseError(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--nope"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestRun_GlobalHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "-h"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage in stdout, got: %s", out.String())
	}
}

func TestRun_GlobalHelpLongFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--help"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected usage in stdout, got: %s", out.String())
	}
}

func TestRun_GlobalHelpViaFlagSetPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", "x", "-h"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got: %s", errBuf.String())
	}
}

func TestRun_MissingDeps(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "version"}, &out, &errBuf, Dependencies{})
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "nope"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
}

func TestRun_Version(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "version"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "dev-vault v") {
		t.Fatalf("unexpected version output: %s", out.String())
	}
}

func TestRun_Help(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestRun_HelpCommandSpecific(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "pull"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "dev-vault") || !strings.Contains(out.String(), "pull") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestRun_HelpCommandSpecific_Version(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "version"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "dev-vault version") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestRun_HelpCommandSpecific_List(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "list"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "list") || !strings.Contains(out.String(), "name-regex") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestRun_HelpCommandSpecific_Push(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "push"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "push") || !strings.Contains(out.String(), "--yes") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestRun_HelpUnknownCommandSpecific(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "help", "nope"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "unknown command for help") {
		t.Fatalf("unexpected stderr: %s", errBuf.String())
	}
}

func TestRun_NoCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage on stderr")
	}
}

func TestRun_EmptyArgv(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage on stderr, got: %s", errBuf.String())
	}
}

func TestRun_SubcommandHelpFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "pull", "-h"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got: %s", errBuf.String())
	}
}

func TestRun_SubcommandHelpFlag_List(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "list", "-h"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got: %s", errBuf.String())
	}
}

func TestRun_SubcommandHelpFlag_Push(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "push", "-h"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got: %s", errBuf.String())
	}
}

func TestRun_SubcommandConfigFlag(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
  "organization_id": "00000000-0000-0000-0000-000000000000",
  "project_id": "11111111-1111-1111-1111-111111111111",
  "region": "fr-par",
  "mapping": {
    "bweb-env-bsmart-dev": {
      "file": ".env.bsmart.rework",
      "format": "raw",
      "type": "opaque",
      "mode": "sync"
    }
  }
}
`)

	api := newFakeSecretAPI()
	api.AddSecret("11111111-1111-1111-1111-111111111111", "bweb-env-bsmart-dev", "/", secret.SecretTypeOpaque)

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		if cfg.ProjectID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("unexpected project id: %s", cfg.ProjectID)
		}
		return api, nil
	})

	// Use subcommand-level --config (this used to fail with "flag provided but not defined").
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "list", "--config", filepath.Join(dir, config.DefaultConfigName), "--json"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "bweb-env-bsmart-dev") {
		t.Fatalf("unexpected list output: %s", out.String())
	}
	if !strings.Contains(errBuf.String(), "mode=sync") {
		t.Fatalf("expected legacy sync warning, got stderr=%q", errBuf.String())
	}
}

func TestRunList_JSONAndTableAndErrors(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)

	api := newFakeSecretAPI()
	api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "b-dev", "/", secret.SecretTypeKeyValue)
	api.AddSecret("proj", "c-prod", "/", secret.SecretTypeOpaque)

	open := func(cfg config.Config, s string) (SecretAPI, error) { return api, nil }
	deps := baseDeps(open)

	t.Run("InvalidRegex", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--name-regex", "["}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--type", "nope"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--json"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(out.Bytes(), &items); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		// Only *-dev secrets should be included.
		if len(items) != 2 {
			t.Fatalf("expected 2, got %d", len(items))
		}
	})

	t.Run("Table", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--name-contains", "a"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "NAME") {
			t.Fatalf("expected table header, got %s", out.String())
		}
		if !strings.Contains(out.String(), "a-dev") {
			t.Fatalf("expected a-dev, got %s", out.String())
		}
	})

	t.Run("ListAPIError", func(t *testing.T) {
		api.listErr = errors.New("boom")
		defer func() { api.listErr = nil }()
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("JSONEncodeError", func(t *testing.T) {
		api.listCalls = 0
		failWriter := &failingWriter{}
		var errBuf bytes.Buffer
		code := runList(commandContext{
			stdout:     failWriter,
			stderr:     &errBuf,
			configPath: cfgPath,
			deps:       deps,
		}, []string{"--json"})
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

type failingWriter struct{}

func (*failingWriter) Write(p []byte) (int, error) { return 0, errors.New("nope") }

type stubSecretAPI struct {
	listFn        func(req ListSecretsInput) ([]*SecretRecord, error)
	accessFn      func(req AccessSecretVersionInput) (*SecretVersionRecord, error)
	createSecret  func(req CreateSecretInput) (*SecretRecord, error)
	createVersion func(req CreateSecretVersionInput) (*SecretVersionRecord, error)
}

func (s *stubSecretAPI) ListSecrets(req ListSecretsInput) ([]*SecretRecord, error) {
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

func TestRunList_MoreBranches(t *testing.T) {
	t.Run("ParseError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := runList(commandContext{
			stdout: &out,
			stderr: &errBuf,
			deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
				return nil, nil
			}),
		}, []string{"--nope"})
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("LoadAndOpenError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := runList(commandContext{
			stdout:     &out,
			stderr:     &errBuf,
			configPath: "/nope.json",
			deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
				return nil, nil
			}),
		}, []string{})
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("ValidRegexFilters", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)

		api := newFakeSecretAPI()
		api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
		api.AddSecret("proj", "b-dev", "/", secret.SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--name-regex", "^a", "--json"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if strings.Contains(out.String(), "b-dev") {
			t.Fatalf("expected b-dev to be filtered out, got %s", out.String())
		}
	})

	t.Run("ValidTypeFilterUsesSingleType", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)

		api := newFakeSecretAPI()
		api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
		api.AddSecret("proj", "b-dev", "/", secret.SecretTypeKeyValue)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		api.listCalls = 0
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--type", "opaque", "--json"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if api.listCalls != 1 {
			t.Fatalf("expected 1 list call, got %d", api.listCalls)
		}
	})

	t.Run("NilSecretAndPathMismatchAreSkipped", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)

		api := &stubSecretAPI{
			listFn: func(req ListSecretsInput) ([]*SecretRecord, error) {
				if req.Type != string(secret.SecretTypeOpaque) {
					return nil, nil
				}
				return []*SecretRecord{
					nil,
					{ID: "s1", ProjectID: "proj", Name: "a-dev", Path: "/other", Type: string(secret.SecretTypeOpaque)},
				}, nil
			},
			accessFn: func(AccessSecretVersionInput) (*SecretVersionRecord, error) {
				return nil, errors.New("not used")
			},
			createSecret: func(CreateSecretInput) (*SecretRecord, error) {
				return nil, errors.New("not used")
			},
			createVersion: func(CreateSecretVersionInput) (*SecretVersionRecord, error) {
				return nil, errors.New("not used")
			},
		}

		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--path", "/wanted", "--json"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if strings.Contains(out.String(), "a-dev") {
			t.Fatalf("expected a-dev to be filtered out by path, got %s", out.String())
		}
	})
}

func TestRunPull_RawAndErrors(t *testing.T) {
	root := t.TempDir()
	cfg := `{
	  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{
    "foo-dev":{"file":"out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"},
    "push-only-dev":{"file":"x","mode":"push","type":"opaque"}
  }
}`
	cfgPath := writeConfig(t, root, cfg)

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte{0, 1, 2})

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	t.Run("ParseError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "--nope"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("AllSelectsOnlyPullable", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "--all"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if !strings.Contains(out.String(), "pulled foo-dev") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})

	t.Run("NonDevNameRefused", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "foo"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("FileExists", func(t *testing.T) {
		// Seed output file so pull errors without --overwrite.
		if err := os.WriteFile(filepath.Join(root, "out.bin"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed: %v", err)
		}
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "foo-dev"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("WriteError", func(t *testing.T) {
		// Make parent a file so AtomicWriteFile fails with mkdirall error.
		if err := os.WriteFile(filepath.Join(root, "notadir"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seed: %v", err)
		}
		// Update mapping in-place by pointing to notadir/out.bin using a new config.
		cfgPath2 := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"notadir/out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath2, "pull", "foo-dev", "--overwrite"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("AccessError", func(t *testing.T) {
		api.accessErr = errors.New("boom")
		defer func() { api.accessErr = nil }()
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "foo-dev", "--overwrite"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("ResolveNotFound", func(t *testing.T) {
		cfgPath2 := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"missing-dev":{"file":"x","path":"/","type":"opaque"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath2, "pull", "missing-dev"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

func TestRunPull_SelectionErrorsAndLoadError(t *testing.T) {
	t.Run("LoadError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", "/nope.json", "pull", "--all"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		}))
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("NoSecretsSpecified", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		api := newFakeSecretAPI()
		sec := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
		api.AddEnabledVersion(sec.ID, []byte{1})
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("AllAndPositional", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		api := newFakeSecretAPI()
		sec := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
		api.AddEnabledVersion(sec.ID, []byte{1})
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "--all", "foo-dev"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("AllSelectsNone", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"out.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
		api := newFakeSecretAPI()
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "--all"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})
}

func TestRunPull_DotenvAndTypeMismatch(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"kv-dev":{"file":"kv.env","format":"dotenv","path":"/","mode":"sync","type":"key_value"}}}`)

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "kv-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte(`{"A":1}`))

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "kv-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPull_DotenvSuccess(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"kv-dev":{"file":"kv.env","format":"dotenv","path":"/","mode":"sync","type":"key_value"}}}`)

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "kv-dev", "/", secret.SecretTypeKeyValue)
	api.AddEnabledVersion(sec.ID, []byte(`{"A":"x","B":1}`))

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "kv-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
	}

	got, err := os.ReadFile(filepath.Join(root, "kv.env"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(got), "A=") || !strings.Contains(string(got), "B=") {
		t.Fatalf("unexpected dotenv file: %q", string(got))
	}
}

func TestRunPush_RawAndDotenvAndCreateMissing(t *testing.T) {
	root := t.TempDir()
	cfg := `{
	  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{
    "foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"},
    "bar-dev":{"file":"bar.env","format":"dotenv","path":"/","mode":"sync","type":"key_value"},
    "new-dev":{"file":"new.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}
  }
}`
	cfgPath := writeConfig(t, root, cfg)

	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bar.env"), []byte("A=1\nB=\"x\"\n"), 0o644); err != nil {
		t.Fatalf("write bar.env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "new.bin"), []byte("N"), 0o644); err != nil {
		t.Fatalf("write new.bin: %v", err)
	}

	api := newFakeSecretAPI()
	foo := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
	bar := api.AddSecret("proj", "bar-dev", "/", secret.SecretTypeKeyValue)
	_ = foo
	_ = bar

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	t.Run("ParseError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "--nope"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("RefuseBatchWithoutYes", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "--all"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("SinglePushRawWithDescription", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		vers := api.versions[foo.ID]
		if len(vers) != 1 || string(vers[0].data) != "DATA" {
			t.Fatalf("unexpected versions: %#v", vers)
		}
	})

	t.Run("DotenvPush", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "bar-dev", "--description", "desc"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		vers := api.versions[bar.ID]
		if len(vers) != 1 {
			t.Fatalf("expected 1 version, got %d", len(vers))
		}
		var m map[string]string
		if err := json.Unmarshal(vers[0].data, &m); err != nil {
			t.Fatalf("unmarshal pushed json: %v", err)
		}
		want := map[string]string{"A": "1", "B": "x"}
		if !reflect.DeepEqual(m, want) {
			t.Fatalf("unexpected json\nwant=%#v\ngot =%#v", want, m)
		}
	})

	t.Run("CreateMissing", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "new-dev", "--create-missing", "--description", "desc"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		// new secret should now exist.
		var created *SecretRecord
		for _, s := range api.secrets {
			if s.Name == "new-dev" {
				created = s
			}
		}
		if created == nil {
			t.Fatalf("expected secret to be created")
		}
	})

	t.Run("CreateMissingRequiresType", func(t *testing.T) {
		cfgPath2 := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"new.bin","mode":"sync"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath2, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

func TestRunPush_MoreBranches(t *testing.T) {
	t.Run("LoadError", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", "/nope.json", "push", "--all", "--yes"}, &out, &errBuf, baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		}))
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("NoSecretsSpecified", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("AllAndPositional", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "--all", "foo-dev"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("DefaultDescriptionUsesHostname", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		foo := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		vers := api.versions[foo.ID]
		if len(vers) != 1 || vers[0].description == nil {
			t.Fatalf("expected description to be set, got %#v", vers)
		}
		if !strings.Contains(*vers[0].description, "dev-vault push") || !strings.Contains(*vers[0].description, "host") {
			t.Fatalf("unexpected description: %q", *vers[0].description)
		}
	})
}

func TestHelpersAndBranches(t *testing.T) {
	// stringSliceFlag.String
	var ss stringSliceFlag
	_ = ss.String()
	_ = ss.Set("a")

	// parseSecretType cases + error
	for _, tt := range []string{"opaque", "certificate", "key_value", "basic_credentials", "database_credentials", "ssh_key"} {
		if _, err := parseSecretType(tt); err != nil {
			t.Fatalf("parseSecretType(%s): %v", tt, err)
		}
	}
	if _, err := parseSecretType("nope"); err == nil {
		t.Fatalf("expected error")
	}

	// selectMappingTargets default-mode and various errors.
	mapping := map[string]config.MappingEntry{
		"a-dev": {File: "a", Mode: "both"},
		"b-dev": {File: "b", Mode: "pull"},
		"c-dev": {File: "c", Mode: "push"},
	}
	if _, err := selectMappingTargets(mapping, true, []string{"a-dev"}, "pull"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectMappingTargets(mapping, false, nil, "pull"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectMappingTargets(mapping, true, nil, "nope"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectMappingTargets(mapping, true, nil, "pull"); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if _, err := selectMappingTargets(mapping, false, []string{"nope"}, "pull"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectMappingTargets(mapping, false, []string{"missing-dev"}, "pull"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := selectMappingTargets(mapping, false, []string{"c-dev"}, "pull"); err == nil {
		t.Fatalf("expected error")
	}

	// jsonToDotenv marshals non-string values as JSON.
	out, err := jsonToDotenv([]byte(`{"A":"x","B":1}`))
	if err != nil {
		t.Fatalf("jsonToDotenv: %v", err)
	}
	if !strings.Contains(string(out), "A=") || !strings.Contains(string(out), "B=") {
		t.Fatalf("unexpected dotenv output: %s", string(out))
	}

	// dotenvToJSON error.
	if _, err := dotenvToJSON([]byte("NOPE")); err == nil {
		t.Fatalf("expected error")
	}
}

func TestOpenScalewaySecretAPI(t *testing.T) {
	t.Run("InvalidRegion", func(t *testing.T) {
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "nope",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("Env", func(t *testing.T) {
		t.Setenv("SCW_ACCESS_KEY", "SCW1234567890ABCDEFG")                 // gitleaks:allow
		t.Setenv("SCW_SECRET_KEY", "00000000-0000-0000-0000-000000000000") // gitleaks:allow
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
		}, "")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("Env_NewClientError", func(t *testing.T) {
		t.Setenv("SCW_ACCESS_KEY", "SCW1234567890ABCDEFG")                 // gitleaks:allow
		t.Setenv("SCW_SECRET_KEY", "00000000-0000-0000-0000-000000000000") // gitleaks:allow
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "not-a-uuid",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "create scaleway client") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ProfileMissingConfig", func(t *testing.T) {
		t.Setenv("SCW_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "p1",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("GetProfileError", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles: {}
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "missing",
		}, "")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "get scaleway profile") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Profile", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles:
  p1:
    access_key: SCW234567890ABCDEFGH # gitleaks:allow
    secret_key: 22222222-2222-2222-2222-222222222222 # gitleaks:allow
    default_organization_id: 22222222-2222-2222-2222-222222222222
    default_project_id: 22222222-2222-2222-2222-222222222222
    default_region: fr-par
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "p1",
		}, "")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("ProfileOverrideWins", func(t *testing.T) {
		cfgPath := filepath.Join(t.TempDir(), "config.yaml")
		yaml := strings.TrimSpace(`
access_key: SCW1234567890ABCDEFG # gitleaks:allow
secret_key: 00000000-0000-0000-0000-000000000000 # gitleaks:allow
default_organization_id: 00000000-0000-0000-0000-000000000000
default_project_id: 00000000-0000-0000-0000-000000000000
default_region: fr-par
profiles:
  p2:
    access_key: SCW234567890ABCDEFGH # gitleaks:allow
    secret_key: 22222222-2222-2222-2222-222222222222 # gitleaks:allow
    default_organization_id: 22222222-2222-2222-2222-222222222222
    default_project_id: 22222222-2222-2222-2222-222222222222
    default_region: fr-par
`) + "\n"
		if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
			t.Fatalf("write scw config: %v", err)
		}
		t.Setenv("SCW_CONFIG_PATH", cfgPath)
		_, err := OpenScalewaySecretAPI(config.Config{
			OrganizationID: "00000000-0000-0000-0000-000000000000",
			ProjectID:      "00000000-0000-0000-0000-000000000000",
			Region:         "fr-par",
			Profile:        "missing",
		}, "p2")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
}

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies("v1", "c1", "d1")
	if deps.Version != "v1" || deps.Commit != "c1" || deps.Date != "d1" {
		t.Fatalf("unexpected deps: %#v", deps)
	}
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil {
		t.Fatalf("expected all funcs set: %#v", deps)
	}
}

func TestReorderFlags(t *testing.T) {
	t.Run("FlagsAfterArgs", func(t *testing.T) {
		got := reorderFlags([]string{"foo-dev", "--overwrite"}, map[string]bool{"overwrite": false})
		want := []string{"--overwrite", "foo-dev"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})

	t.Run("ValueFlagSeparateArg", func(t *testing.T) {
		got := reorderFlags([]string{"foo-dev", "--description", "d"}, map[string]bool{"description": true})
		want := []string{"--description", "d", "foo-dev"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})

	t.Run("ValueFlagEquals", func(t *testing.T) {
		got := reorderFlags([]string{"foo-dev", "--description=d"}, map[string]bool{"description": true})
		want := []string{"--description=d", "foo-dev"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})

	t.Run("DoubleDashStopsParsing", func(t *testing.T) {
		got := reorderFlags([]string{"--description", "d", "--", "foo-dev", "--overwrite"}, map[string]bool{"description": true, "overwrite": false})
		want := []string{"--description", "d", "foo-dev", "--overwrite"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})

	t.Run("HyphenIsPositional", func(t *testing.T) {
		got := reorderFlags([]string{"-", "--json"}, map[string]bool{"json": false})
		want := []string{"--json", "-"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})

	t.Run("MissingValueDoesNotPanic", func(t *testing.T) {
		got := reorderFlags([]string{"--description"}, map[string]bool{"description": true})
		want := []string{"--description"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want=%#v got=%#v", want, got)
		}
	})
}

func TestLoadAndOpenAPI_GetwdError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	deps.Getwd = func() (string, error) { return "", errors.New("boom") }
	_, _, err := loadAndOpenAPI("", "", deps)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPI_Success(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"sync"}}}`)

	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	loaded, gotAPI, err := loadAndOpenAPI(cfgPath, "", deps)
	if err != nil || loaded == nil || gotAPI == nil {
		t.Fatalf("expected success, got err=%v loaded=%v api=%v", err, loaded, gotAPI)
	}
}

func TestLoadAndOpenAPI_ConfigError(t *testing.T) {
	_, _, err := loadAndOpenAPI("/nope.json", "", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPI_OpenError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)
	_, _, err := loadAndOpenAPI(cfgPath, "", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, errors.New("boom")
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveSecretByNameAndPath_MultipleMatches(t *testing.T) {
	api := newFakeSecretAPI()
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	_, err := resolveSecretByNameAndPath(api, config.Config{ProjectID: "proj", Region: "fr-par"}, "dup-dev", "/")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestResolveSecretByNameAndPath_NotFound(t *testing.T) {
	api := newFakeSecretAPI()
	_, err := resolveSecretByNameAndPath(api, config.Config{ProjectID: "proj", Region: "fr-par"}, "missing-dev", "/")
	var nf *notFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected notFoundError, got %v", err)
	}
	_ = nf.Error()
}

func TestListSecretsByTypes_Error(t *testing.T) {
	api := newFakeSecretAPI()
	api.listErr = errors.New("boom")
	_, err := listSecretsByTypes(api, ListSecretsInput{ProjectID: "p"}, []string{string(secret.SecretTypeOpaque)})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunPush_DefaultDescriptionAndHostnameErrorAndVersionError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}

	api := newFakeSecretAPI()
	foo := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
	_ = foo

	deps := Dependencies{
		Version: "v", Commit: "c", Date: "d",
		OpenSecretAPI: func(cfg config.Config, s string) (SecretAPI, error) { return api, nil },
		Now:           func() time.Time { return time.Unix(0, 0) },
		Hostname:      func() (string, error) { return "", errors.New("nope") },
		Getwd:         os.Getwd,
	}

	api.createVerErr = errors.New("boom")
	defer func() { api.createVerErr = nil }()
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "create version") {
		t.Fatalf("expected error message, got %s", errBuf.String())
	}
}

func TestRunPull_DotenvFormatError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"kv-dev":{"file":"kv.env","format":"dotenv","path":"/","mode":"sync","type":"key_value"}}}`)
	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "kv-dev", "/", secret.SecretTypeKeyValue)
	api.AddEnabledVersion(sec.ID, []byte("not-json"))

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "kv-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_CreateMissingInvalidMappingTypeAndCreateSecretError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	t.Run("InvalidMappingType", func(t *testing.T) {
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"nope"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("CreateSecretError", func(t *testing.T) {
		api.createSecretErr = errors.New("boom")
		defer func() { api.createSecretErr = nil }()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

func TestRunPush_ResolveErrorNoCreateMissing(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_FileReadError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"missing.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPull_MappingResolveError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"../oops","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "x-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_MappingResolveError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"../oops","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestListCommand_UsesAllTypesWhenNoTypeFilter(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "b-dev", "/", secret.SecretTypeKeyValue)

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--json"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	// listSecretsByTypes should call ListSecrets once per type.
	if api.listCalls < len(supportedSecretTypes()) {
		t.Fatalf("expected >= %d list calls, got %d", len(supportedSecretTypes()), api.listCalls)
	}
}

func TestRunList_LoadAndOpenViaDiscovery(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)
	_ = cfgPath
	api := newFakeSecretAPI()
	api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(filepath.Join(root, "nested")); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "list"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
	}
}

func TestRunPush_DisablePrevious(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}

	api := newFakeSecretAPI()
	foo := api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	// Push twice: second disables previous.
	var out1, err1 bytes.Buffer
	if code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "d1"}, &out1, &err1, deps); code != 0 {
		t.Fatalf("first push: %d (%s)", code, err1.String())
	}
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("B"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	var out2, err2 bytes.Buffer
	if code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--disable-previous", "--description", "d2"}, &out2, &err2, deps); code != 0 {
		t.Fatalf("second push: %d (%s)", code, err2.String())
	}
	vers := api.versions[foo.ID]
	if len(vers) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(vers))
	}
	if vers[0].enabled {
		t.Fatalf("expected rev1 disabled")
	}
}

func TestRunPull_ResolveMultipleMatches(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"dup-dev":{"file":"out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "dup-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "dup-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPull_ListErrorViaResolve(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"out.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.listErr = errors.New("boom")
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "foo-dev", "--overwrite"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_DotenvParseError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.env","format":"dotenv","path":"/","mode":"sync","type":"key_value"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.env"), []byte("NOPE"), 0o644); err != nil {
		t.Fatalf("write in.env: %v", err)
	}
	api := newFakeSecretAPI()
	api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeKeyValue)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_TypeMismatch(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"key_value"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	api.AddSecret("proj", "foo-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPush_CreateMissing_ResolveStillFails(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"sync","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	// Override CreateSecret to succeed but not persist, forcing resolve to still fail.
	api2 := *api
	api2.createSecretErr = nil
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return &api2, nil
	})
	// Monkey patch method by embedding a wrapper.
	wrapped := &createSecretNoPersist{inner: &api2}
	deps.OpenSecretAPI = func(cfg config.Config, s string) (SecretAPI, error) { return wrapped, nil }

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

type createSecretNoPersist struct{ inner *fakeSecretAPI }

func (c *createSecretNoPersist) ListSecrets(req ListSecretsInput) ([]*SecretRecord, error) {
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
