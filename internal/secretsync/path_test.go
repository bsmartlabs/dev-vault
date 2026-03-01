package secretsync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveFile(t *testing.T) {
	t.Run("RootRequired", func(t *testing.T) {
		if _, err := resolveFile("", "x"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("RelativeRequired", func(t *testing.T) {
		if _, err := resolveFile(t.TempDir(), ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("RejectAbsolute", func(t *testing.T) {
		if _, err := resolveFile(t.TempDir(), "/tmp/x"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("RejectEscape", func(t *testing.T) {
		if _, err := resolveFile(t.TempDir(), "../x"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ResolveUnderRoot", func(t *testing.T) {
		root := t.TempDir()
		got, err := resolveFile(root, "a/b.txt")
		if err != nil {
			t.Fatalf("resolveFile: %v", err)
		}
		if !strings.HasPrefix(got, root+string(filepath.Separator)) {
			t.Fatalf("expected resolved path under root: %s", got)
		}
	})

	t.Run("ResolveRelativeRoot", func(t *testing.T) {
		cwd := t.TempDir()
		realCwd, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			t.Fatalf("eval symlinks cwd: %v", err)
		}
		orig, err := filepath.Abs(".")
		if err != nil {
			t.Fatalf("abs cwd: %v", err)
		}
		if chdirErr := os.Chdir(cwd); chdirErr != nil {
			t.Fatalf("chdir: %v", chdirErr)
		}
		t.Cleanup(func() {
			_ = os.Chdir(orig)
		})

		got, err := resolveFile(".", "x")
		if err != nil {
			t.Fatalf("resolveFile relative root: %v", err)
		}
		if !strings.HasPrefix(got, realCwd+string(filepath.Separator)) {
			t.Fatalf("expected resolved path under cwd root: %s", got)
		}
	})
}
