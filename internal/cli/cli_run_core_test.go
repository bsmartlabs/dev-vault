package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

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
