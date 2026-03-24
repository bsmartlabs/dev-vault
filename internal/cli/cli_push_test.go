package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/pathpolicy"
)

func TestRunPushRawAndDotenvAndCreateMissing(t *testing.T) {
	root := t.TempDir()
	cfg := `{
	  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{
    "foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"},
    "bar-dev":{"file":"bar.env","format":"dotenv","path":"/","mode":"push","type":"key_value"},
    "new-dev":{"file":"new.bin","format":"raw","path":"/","mode":"push","type":"opaque"}
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
	foo := api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
	bar := api.AddSecret("proj", "bar-dev", "/", SecretTypeKeyValue)
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
		vers := api.Versions[foo.ID]
		if len(vers) != 1 || string(vers[0].Data) != "DATA" {
			t.Fatalf("unexpected versions: %#v", vers)
		}
	})

	t.Run("DotenvPush", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "bar-dev", "--description", "desc"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		vers := api.Versions[bar.ID]
		if len(vers) != 1 {
			t.Fatalf("expected 1 version, got %d", len(vers))
		}
		var m map[string]string
		if err := json.Unmarshal(vers[0].Data, &m); err != nil {
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
		for i := range api.Secrets {
			if api.Secrets[i].Name == "new-dev" {
				created = &api.Secrets[i]
			}
		}
		if created == nil {
			t.Fatalf("expected secret to be created")
		}
	})

	t.Run("CreateMissingRequiresType", func(t *testing.T) {
		cfgPath2 := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"new.bin","mode":"push"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath2, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

func TestRunPushMoreBranches(t *testing.T) {
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
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("AllAndPositional", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "--all", "foo-dev"}, &out, &errBuf, deps)
		if code != 2 {
			t.Fatalf("expected 2, got %d", code)
		}
	})

	t.Run("DefaultDescriptionUsesHostname", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
		if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
			t.Fatalf("write in.bin: %v", err)
		}
		api := newFakeSecretAPI()
		foo := api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
		deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev"}, &out, &errBuf, deps)
		if code != 0 {
			t.Fatalf("expected 0, got %d (%s)", code, errBuf.String())
		}
		vers := api.Versions[foo.ID]
		if len(vers) != 1 || vers[0].Description == nil {
			t.Fatalf("expected description to be set, got %#v", vers)
		}
		if !strings.Contains(*vers[0].Description, "dev-vault push") || !strings.Contains(*vers[0].Description, "host") {
			t.Fatalf("unexpected description: %q", *vers[0].Description)
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
	mapping := map[string]mapping.Entry{
		"a-dev": {File: "a", Mode: "push"},
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
	} else if !strings.Contains(err.Error(), "mapping entry c-dev has mode=push, cannot be used with pull") {
		t.Fatalf("unexpected mode mismatch error: %v", err)
	}

	// jsonToDotenv rejects non-string values.
	if _, err := jsonToDotenvForTest([]byte(`{"A":"x","B":1}`)); err == nil {
		t.Fatal("expected error")
	}

	// dotenvToJSON error.
	if _, err := dotenvToJSONForTest([]byte("NOPE")); err == nil {
		t.Fatalf("expected error")
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

	t.Run("DoubleDashPreservedForDashPrefixedPositional", func(t *testing.T) {
		got := reorderFlags([]string{"--", "--config-dev"}, map[string]bool{"overwrite": false})
		want := []string{"--", "--config-dev"}
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


func TestRunPushDefaultDescriptionAndHostnameErrorAndVersionError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}

	api := newFakeSecretAPI()
	foo := api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
	_ = foo

	deps := Dependencies{
		Version: "v", Commit: "c", Date: "d",
		OpenSecretAPI:      func(cfg config.Config, s string) (SecretAPI, error) { return api, nil },
		Now:                func() time.Time { return time.Unix(0, 0) },
		Hostname:           func() (string, error) { return "", errors.New("nope") },
		ResolveProjectPath: pathpolicy.ResolveProjectFile,
	}

	api.CreateVerErr = errors.New("boom")
	defer func() { api.CreateVerErr = nil }()
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "create version") {
		t.Fatalf("expected error message, got %s", errBuf.String())
	}
}

func TestRunPushCreateMissingInvalidMappingTypeAndCreateSecretError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	t.Run("InvalidMappingType", func(t *testing.T) {
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"nope"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})

	t.Run("CreateSecretError", func(t *testing.T) {
		api.CreateSecretErr = errors.New("boom")
		defer func() { api.CreateSecretErr = nil }()
		cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
		var out, errBuf bytes.Buffer
		code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--create-missing"}, &out, &errBuf, deps)
		if code != 1 {
			t.Fatalf("expected 1, got %d", code)
		}
	})
}

func TestRunPushResolveErrorNoCreateMissing(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
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

func TestRunPushFileReadError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"missing.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPushMappingResolveError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"../oops","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPushDisablePrevious(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}

	api := newFakeSecretAPI()
	foo := api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
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
	vers := api.Versions[foo.ID]
	if len(vers) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(vers))
	}
	if vers[0].Enabled {
		t.Fatalf("expected rev1 disabled")
	}
}

func TestRunPushListErrorViaLookupIndex(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	api.ListErr = errors.New("boom")
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPushDotenvParseError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.env","format":"dotenv","path":"/","mode":"push","type":"key_value"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.env"), []byte("NOPE"), 0o644); err != nil {
		t.Fatalf("write in.env: %v", err)
	}
	api := newFakeSecretAPI()
	api.AddSecret("proj", "foo-dev", "/", SecretTypeKeyValue)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPushTypeMismatch(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"foo-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"key_value"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	api.AddSecret("proj", "foo-dev", "/", SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "foo-dev", "--description", "desc"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunPushCreateMissingResolveStillFails(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write in.bin: %v", err)
	}
	api := newFakeSecretAPI()
	// Override CreateSecret to succeed but not persist, forcing resolve to still fail.
	api2 := *api
	api2.CreateSecretErr = nil
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
