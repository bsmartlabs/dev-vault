package cli

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
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

func TestLoadAndOpenAPIConfigDiscoveryError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return nil, nil })
	_, err := loadConfig("", deps)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadConfigAbsolutePath(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)

	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return nil, nil })

	loaded, err := loadConfig(cfgPath, deps)
	if err != nil {
		t.Fatalf("loadConfig absolute path: %v", err)
	}
	if loaded == nil || loaded.Path == "" {
		t.Fatalf("expected loaded config, got %#v", loaded)
	}
}

func TestLoadConfigAbsolutePathLoadError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	_, err := loadConfig("/no/such/config.json", deps)
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestLoadConfigRelativePathBranches(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		})
		old, _ := os.Getwd()
		defer func() { _ = os.Chdir(old) }()
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}

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
		old, _ := os.Getwd()
		defer func() { _ = os.Chdir(old) }()
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		if _, err := loadConfig("missing.json", deps); err == nil {
			t.Fatal("expected load error for missing relative config")
		}
	})
}

func TestLoadAndOpenAPISuccess(t *testing.T) {
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

func TestLoadAndOpenAPIConfigError(t *testing.T) {
	_, err := loadConfig("/nope.json", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPIOpenError(t *testing.T) {
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

func TestLoadProjectConfigBranches(t *testing.T) {
	t.Run("AbsolutePath", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par"}`)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return nil, nil })

		loaded, err := loadProjectConfig(cfgPath, deps)
		if err != nil {
			t.Fatalf("loadProjectConfig absolute path: %v", err)
		}
		if loaded == nil || loaded.Path == "" {
			t.Fatalf("expected loaded config, got %#v", loaded)
		}
	})

	t.Run("DiscoveryError", func(t *testing.T) {
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return nil, nil })
		if _, err := loadProjectConfig("", deps); err == nil {
			t.Fatal("expected discovery error")
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
		old, _ := os.Getwd()
		defer func() { _ = os.Chdir(old) }()
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		if _, err := loadProjectConfig("missing.json", deps); err == nil {
			t.Fatal("expected load error")
		}
	})
}

func TestRunProfileOverridePropagatesToOpenSecretAPI(t *testing.T) {
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

func TestRunReportsServiceInitErrorWhenAPIIsNil(t *testing.T) {
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
	loader, err := configLoaderForPolicy(commandConfigProjectOnly)
	if err != nil || loader == nil {
		t.Fatal("expected project-only loader")
	}
	if _, err := configLoaderForPolicy(commandConfigPolicy(999)); err == nil {
		t.Fatal("expected unsupported policy error")
	}
}

func TestCommandRuntimeInvalidConfigPolicy(t *testing.T) {
	ctx := commandContext{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		deps:   baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return nil, nil }),
	}
	parsed := &parsedCommand{
		fs:           flag.NewFlagSet("test", flag.ContinueOnError),
		configPolicy: commandConfigPolicy(999),
	}
	runtime := newCommandRuntime(ctx, parsed)

	_, err := runtime.prepareResources(parsed.configPolicy)
	if err == nil {
		t.Fatal("expected prepareResources to fail for invalid policy")
	}

	code := runtime.writeStderrError(err)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}

	errText := ctx.stderr.(*bytes.Buffer).String()
	if !strings.Contains(errText, "unsupported command config policy") {
		t.Fatalf("expected unsupported policy error, got %q", errText)
	}
}
