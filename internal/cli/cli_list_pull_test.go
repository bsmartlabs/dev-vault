package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

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
			listFn: func(req ListSecretsInput) ([]SecretRecord, error) {
				if req.Type != SecretTypeOpaque {
					return nil, nil
				}
				return []SecretRecord{
					{ID: "s1", ProjectID: "proj", Name: "a-dev", Path: "/other", Type: SecretTypeOpaque},
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
