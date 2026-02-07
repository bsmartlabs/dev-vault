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
		oldAbs := absFn
		absFn = func(string) (string, error) { return "", errors.New("boom") }
		defer func() { absFn = oldAbs }()
		_, err := FindConfigPath(".")
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
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`), 0o644); err != nil {
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
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`), 0o644); err != nil {
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
			{"BadMode", `{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x","mode":"nope"}}}`, "invalid mode"},
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
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		loaded, err := Load(dir, cfgPath)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		ent := loaded.Cfg.Mapping["a-dev"]
		if ent.Format != "raw" || ent.Path != "/" || ent.Mode != "sync" {
			t.Fatalf("defaults not applied: %+v", ent)
		}
	})

	t.Run("DiscoverySuccess", func(t *testing.T) {
		root := t.TempDir()
		cfgPath := filepath.Join(root, DefaultConfigName)
		if err := os.WriteFile(cfgPath, []byte(`{"organization_id":"o","project_id":"p","region":"fr-par","mapping":{"a-dev":{"file":"x"}}}`), 0o644); err != nil {
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
		oldAbs := absFn
		absFn = func(string) (string, error) { return "", errors.New("boom") }
		defer func() { absFn = oldAbs }()
		_, err := Load(".", DefaultConfigName)
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

	t.Run("AbsRootErrorViaMissingCwd", func(t *testing.T) {
		oldAbs := absFn
		absFn = func(string) (string, error) { return "", errors.New("boom") }
		defer func() { absFn = oldAbs }()
		_, err := ResolveFile(".", "x")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs rootDir") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("AbsJoinedPathError", func(t *testing.T) {
		oldAbs := absFn
		calls := 0
		absFn = func(s string) (string, error) {
			calls++
			if calls == 2 {
				return "", errors.New("boom")
			}
			return oldAbs(s)
		}
		defer func() { absFn = oldAbs }()

		_, err := ResolveFile(t.TempDir(), "x")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs joined path") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("RelPathError", func(t *testing.T) {
		oldRel := relFn
		relFn = func(string, string) (string, error) { return "", errors.New("boom") }
		defer func() { relFn = oldRel }()

		_, err := ResolveFile(t.TempDir(), "x")
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "rel path") {
			t.Fatalf("unexpected error: %v", err)
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
