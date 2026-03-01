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
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

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
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

		api := newFakeSecretAPI()
		api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
		api.AddSecret("proj", "b-dev", "/", secret.SecretTypeKeyValue)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		api.ListCalls = 0
		code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--type", "opaque", "--json"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		if api.ListCalls != 1 {
			t.Fatalf("expected 1 list call, got %d", api.ListCalls)
		}
	})

	t.Run("NilSecretAndPathMismatchAreSkipped", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

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

func TestListCommand_UsesAllTypesWhenNoTypeFilter(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "a-dev", "/", secret.SecretTypeOpaque)
	api.AddSecret("proj", "b-dev", "/", secret.SecretTypeKeyValue)

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "list", "--json"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	// List command should use a single filtered API query for untyped lists.
	if api.ListCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", api.ListCalls)
	}
}

func TestRunList_LoadAndOpenViaDiscovery(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
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
