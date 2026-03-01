package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindConfigPath(t *testing.T) {
	t.Run("EmptyStartDir", func(t *testing.T) {
		_, err := FindConfigPath("")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		dir := t.TempDir()
		_, err := FindConfigPath(dir)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("AbsErrorViaMissingCwd", func(t *testing.T) {
		deps := defaultConfigDeps
		deps.abs = func(string) (string, error) { return "", errors.New("boom") }
		_, err := findConfigPath(".", deps)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs startDir") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("FindsUpwards", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := filepath.Join(root, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull"}}}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		nested := filepath.Join(root, "a", "b", "c")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		found, err := FindConfigPath(nested)
		if err != nil {
			t.Fatalf("expected config, got error: %v", err)
		}
		if found != cfgPath {
			t.Fatalf("expected %s, got %s", cfgPath, found)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("EmptyStartDir", func(t *testing.T) {
		_, err := Load("", "")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ExplicitRelative", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull"}}}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		loaded, err := Load(dir, DefaultConfigName)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if loaded.Root != dir {
			t.Fatalf("expected root %s, got %s", dir, loaded.Root)
		}
		if loaded.Path != cfgPath {
			t.Fatalf("expected path %s, got %s", cfgPath, loaded.Path)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := Load(dir, cfgPath)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("UnknownField", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}},"nope":1}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := Load(dir, cfgPath)
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("TrailingJSONRejected", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		payload := `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}{"extra":true}`
		if err := os.WriteFile(cfgPath, []byte(payload), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := Load(dir, cfgPath)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "trailing data") {
			t.Fatalf("expected trailing data error, got %v", err)
		}
	})

	t.Run("TrailingJSONSyntaxErrorRejected", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		payload := `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}{`
		if err := os.WriteFile(cfgPath, []byte(payload), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		_, err := Load(dir, cfgPath)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "trailing data") {
			t.Fatalf("expected trailing data error, got %v", err)
		}
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		cases := []struct {
			name    string
			json    string
			wantSub string
		}{
			{"MissingOrg", `{"project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`, "organization_id"},
			{"MissingProject", `{"organization_id":"o","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`, "project_id"},
			{"MissingRegion", `{"organization_id":"o","project_id":"p","mapping":{"a-dev":{"file":"x"}}}`, "region"},
			{"MissingMapping", `{"organization_id":"o","project_id":"p","region":"fr-par"}`, "mapping"},
			{"EmptyMapping", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{}}`, "mapping is empty"},
			{"NonDevKey", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a":{"file":"x"}}}`, "must end with -dev"},
			{"EmptyFile", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":""}}}`, "missing required field: file"},
			{"AbsFile", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"/tmp/x"}}}`, "file must be relative"},
			{"BadFormat", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","format":"nope"}}}`, "invalid format"},
			{"BadPath", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","path":"nope"}}}`, "path must start"},
			{"MissingMode", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`, "missing required field: mode"},
			{"BadMode", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"nope"}}}`, "invalid mode"},
			{"BadType", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull","type":"nope"}}}`, "invalid type"},
			{"BadTypeWhitespace", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull","type":"opaque "}}}`, "invalid type"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				dir := t.TempDir()
				cfgPath := filepath.Join(dir, DefaultConfigName)
				if err := os.WriteFile(cfgPath, []byte(tc.json), 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
				_, err := Load(dir, cfgPath)
				if err == nil {
					t.Fatalf("expected error")
				}
				if !strings.Contains(err.Error(), tc.wantSub) {
					t.Fatalf("expected error containing %q, got %v", tc.wantSub, err)
				}
			})
		}
	})

	t.Run("DefaultsApplied", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull"}}}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		loaded, err := Load(dir, cfgPath)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		ent := loaded.Cfg.Mapping["a-dev"]
		if ent.Format != MappingFormatRaw || ent.Path != "/" || ent.Mode != MappingModePull {
			t.Fatalf("normalization not applied: %+v", ent)
		}
	})

	t.Run("DiscoverySuccess", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := filepath.Join(root, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"pull"}}}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		nested := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		loaded, err := Load(nested, "")
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if loaded.Path != cfgPath {
			t.Fatalf("expected %s, got %s", cfgPath, loaded.Path)
		}
	})

	t.Run("DiscoveryNotFound", func(t *testing.T) {
		dir := t.TempDir()
		_, err := Load(dir, "")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ReadFileError", func(t *testing.T) {
		dir := t.TempDir()
		_, err := Load(dir, filepath.Join(dir, "missing.json"))
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "read config") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("AbsConfigPathErrorViaMissingCwd", func(t *testing.T) {
		deps := defaultConfigDeps
		deps.abs = func(string) (string, error) { return "", errors.New("boom") }
		_, err := loadWithDeps(".", DefaultConfigName, deps)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs config path") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResolveFile(t *testing.T) {
	t.Run("Errors", func(t *testing.T) {
		if _, err := ResolveFile("", "x"); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := ResolveFile("root", ""); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := ResolveFile("root", "/abs"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("EscapesRoot", func(t *testing.T) {
		root := t.TempDir()
		_, err := ResolveFile(root, "../x")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("Success", func(t *testing.T) {
		root := t.TempDir()
		p, err := ResolveFile(root, "a/b.txt")
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if !strings.HasPrefix(p, root+string(filepath.Separator)) {
			t.Fatalf("expected path under root, got %s", p)
		}
	})
}

func TestMappingMode_Allows(t *testing.T) {
	cases := []struct {
		mode       MappingMode
		pull, push bool
	}{
		{mode: MappingModePull, pull: true, push: false},
		{mode: MappingModePush, pull: false, push: true},
		{mode: MappingMode(""), pull: false, push: false},
		{mode: MappingMode("nope"), pull: false, push: false},
	}

	for _, tc := range cases {
		if tc.mode.AllowsPull() != tc.pull {
			t.Fatalf("AllowsPull mismatch for %q", tc.mode)
		}
		if tc.mode.AllowsPush() != tc.push {
			t.Fatalf("AllowsPush mismatch for %q", tc.mode)
		}
	}
}

func TestDevSecretNameWrappers(t *testing.T) {
	if !IsDevSecretName("x-dev") {
		t.Fatal("expected x-dev to be accepted")
	}
	if IsDevSecretName("x-prod") {
		t.Fatal("expected x-prod to be rejected")
	}
	if err := ValidateDevSecretName("x-dev"); err != nil {
		t.Fatalf("expected validation success, got %v", err)
	}
	if err := ValidateDevSecretName("x-prod"); err == nil {
		t.Fatal("expected validation error for non-dev name")
	}
}
