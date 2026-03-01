package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies("v1", "c1", "d1", func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return nil, nil
	})
	if deps.Version != "v1" || deps.Commit != "c1" || deps.Date != "d1" {
		t.Fatalf("unexpected deps: %#v", deps)
	}
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil {
		t.Fatalf("expected all funcs set: %#v", deps)
	}
}

func TestLoadAndOpenAPI_GetwdError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	deps.Getwd = func() (string, error) { return "", errors.New("boom") }
	_, err := loadConfig("", deps)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadConfig_AbsolutePathSkipsGetwd(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	deps.Getwd = func() (string, error) {
		return "", errors.New("getwd should not be called for absolute config path")
	}

	loaded, err := loadConfig(cfgPath, deps)
	if err != nil {
		t.Fatalf("loadConfig absolute path: %v", err)
	}
	if loaded == nil || loaded.Path == "" {
		t.Fatalf("expected loaded config, got %#v", loaded)
	}
}

func TestLoadConfig_AbsolutePathLoadError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	_, err := loadConfig("/no/such/config.json", deps)
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestLoadConfig_RelativePathBranches(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		deps.Getwd = func() (string, error) { return root, nil }

		loaded, err := loadConfig(config.DefaultConfigName, deps)
		if err != nil {
			t.Fatalf("loadConfig relative success: %v", err)
		}
		if loaded == nil {
			t.Fatal("expected loaded config")
		}
	})

	t.Run("LoadError", func(t *testing.T) {
		root := t.TempDir()
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		deps.Getwd = func() (string, error) { return root, nil }
		if _, err := loadConfig("missing.json", deps); err == nil {
			t.Fatal("expected load error for missing relative config")
		}
	})
}

func TestLoadAndOpenAPI_Success(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	loaded, err := loadConfig(cfgPath, deps)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	gotAPI, err := openAPIForLoaded(loaded, "", deps)
	if err != nil || loaded == nil || gotAPI == nil {
		t.Fatalf("expected success, got err=%v loaded=%v api=%v", err, loaded, gotAPI)
	}
}

func TestLoadAndOpenAPI_ConfigError(t *testing.T) {
	_, err := loadConfig("/nope.json", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPI_OpenError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, errors.New("boom")
	})
	loaded, err := loadConfig(cfgPath, deps)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	_, err = openAPIForLoaded(loaded, "", deps)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadProjectConfig_Branches(t *testing.T) {
	t.Run("AbsolutePathSkipsGetwd", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par"}`)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		deps.Getwd = func() (string, error) {
			return "", errors.New("getwd should not be called for absolute config path")
		}

		loaded, err := loadProjectConfig(cfgPath, deps)
		if err != nil {
			t.Fatalf("loadProjectConfig absolute path: %v", err)
		}
		if loaded == nil || loaded.Path == "" {
			t.Fatalf("expected loaded config, got %#v", loaded)
		}
	})

	t.Run("GetwdError", func(t *testing.T) {
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		deps.Getwd = func() (string, error) { return "", errors.New("boom") }
		if _, err := loadProjectConfig("", deps); err == nil {
			t.Fatal("expected getwd error")
		}
	})

	t.Run("AbsolutePathLoadError", func(t *testing.T) {
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		if _, err := loadProjectConfig("/no/such/project-config.json", deps); err == nil {
			t.Fatal("expected load error")
		}
	})

	t.Run("RelativePathLoadError", func(t *testing.T) {
		root := t.TempDir()
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		deps.Getwd = func() (string, error) { return root, nil }
		if _, err := loadProjectConfig("missing.json", deps); err == nil {
			t.Fatal("expected load error")
		}
	})
}

func TestRun_ProfileOverridePropagatesToOpenSecretAPI(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)

	var gotProfile string
	deps := baseDeps(func(cfg config.Config, profile string) (SecretAPI, error) {
		gotProfile = profile
		return newFakeSecretAPI(), nil
	})

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "--profile", "ci-prof", "list"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, errBuf.String())
	}
	if gotProfile != "ci-prof" {
		t.Fatalf("expected profile override to propagate, got %q", gotProfile)
	}
	if !strings.Contains(out.String(), "NAME") || !strings.Contains(out.String(), "TYPE") {
		t.Fatalf("unexpected list output: %s", out.String())
	}
}

func TestRun_ReportsServiceInitErrorWhenAPIIsNil(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)
	deps := baseDeps(func(cfg config.Config, profile string) (SecretAPI, error) {
		return nil, nil
	})

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "list"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected init service error, got %q", errBuf.String())
	}
}

func TestCommandErrorExitCodeContract(t *testing.T) {
	base := errors.New("boom")
	if code := exitCodeForError(nil); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if code := exitCodeForError(helpError(base)); code != 0 {
		t.Fatalf("expected 0 for help, got %d", code)
	}
	if code := exitCodeForError(usageError(base)); code != 2 {
		t.Fatalf("expected 2 for usage, got %d", code)
	}
	if code := exitCodeForError(runtimeError(base)); code != 1 {
		t.Fatalf("expected 1 for runtime, got %d", code)
	}
	if code := exitCodeForError(outputError(base)); code != 1 {
		t.Fatalf("expected 1 for output, got %d", code)
	}
}

func TestOpenScalewaySecretAPIWrapper(t *testing.T) {
	_, err := OpenScalewaySecretAPI(config.Config{
		OrganizationID: "00000000-0000-0000-0000-000000000000",
		ProjectID:      "00000000-0000-0000-0000-000000000000",
		Region:         "bad-region",
	}, "")
	if err == nil {
		t.Fatalf("expected wrapper to propagate open error")
	}
}

func TestConfigPolicyContracts(t *testing.T) {
	if _, err := configLoaderForPolicy(commandConfigNone); err == nil {
		t.Fatal("expected unsupported policy error for commandConfigNone")
	}

	var errBuf bytes.Buffer
	runtime := commandRuntime{
		ctx: commandContext{
			stdout: &bytes.Buffer{},
			stderr: &errBuf,
			deps: baseDeps(func(cfg config.Config, profile string) (SecretAPI, error) {
				return newFakeSecretAPI(), nil
			}),
		},
	}
	if code := runtime.executeWithConfigPolicy(commandConfigNone, func(loaded *config.Loaded, service secretsync.Service) error { return nil }); code != 1 {
		t.Fatalf("expected runtime error exit code, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "unsupported command config policy") {
		t.Fatalf("unexpected stderr: %q", errBuf.String())
	}

	errBuf.Reset()
	if code := runtime.runMappingCommand(commandConfigNone, mapping.ModePull, true, nil, func(service secretsync.Service, targets []mapping.Target) error { return nil }); code != 1 {
		t.Fatalf("expected runtime error exit code, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "unsupported command config policy") {
		t.Fatalf("unexpected stderr: %q", errBuf.String())
	}
}
