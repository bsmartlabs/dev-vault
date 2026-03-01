package pathpolicy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveProjectFile(t *testing.T) {
	t.Run("Errors", func(t *testing.T) {
		if _, err := ResolveProjectFile("", "x"); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := ResolveProjectFile("root", ""); err == nil {
			t.Fatalf("expected error")
		}
		if _, err := ResolveProjectFile("root", "/abs"); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("AbsRootError", func(t *testing.T) {
		deps := defaultResolverDeps
		deps.abs = func(string) (string, error) { return "", errors.New("boom") }
		_, err := resolveProjectFileWithDeps(".", "x", deps)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs rootDir") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("AbsJoinedPathError", func(t *testing.T) {
		oldAbs := defaultResolverDeps.abs
		calls := 0
		deps := defaultResolverDeps
		deps.abs = func(s string) (string, error) {
			calls++
			if calls == 2 {
				return "", errors.New("boom")
			}
			return oldAbs(s)
		}
		_, err := resolveProjectFileWithDeps(t.TempDir(), "x", deps)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "abs joined path") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("RelPathError", func(t *testing.T) {
		deps := defaultResolverDeps
		deps.rel = func(string, string) (string, error) { return "", errors.New("boom") }
		_, err := resolveProjectFileWithDeps(t.TempDir(), "x", deps)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "rel path") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("EscapesRoot", func(t *testing.T) {
		root := t.TempDir()
		_, err := ResolveProjectFile(root, "../x")
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("SuccessAbsoluteRoot", func(t *testing.T) {
		root := t.TempDir()
		p, err := ResolveProjectFile(root, "a/b.txt")
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if !strings.HasPrefix(p, root+string(filepath.Separator)) {
			t.Fatalf("expected path under root, got %s", p)
		}
	})

	t.Run("SuccessRelativeRoot", func(t *testing.T) {
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
		p, err := ResolveProjectFile(".", "a/b.txt")
		if err != nil {
			t.Fatalf("resolve relative root: %v", err)
		}
		if !strings.HasPrefix(p, realCwd+string(filepath.Separator)) {
			t.Fatalf("expected path under cwd root, got %s", p)
		}
	})
}
