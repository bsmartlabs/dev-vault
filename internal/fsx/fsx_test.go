package fsx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWriteFile_Success(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")
	data := []byte("hello")

	if err := AtomicWriteFile(dest, data, 0o600, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("unexpected contents: %q", string(got))
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected perm 0600, got %o", info.Mode().Perm())
	}
}

func TestAtomicWriteFile_Exists(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.txt")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	err := AtomicWriteFile(dest, []byte("y"), 0o600, false)
	if !errors.Is(err, ErrExists) {
		t.Fatalf("expected ErrExists, got %v", err)
	}
}

func TestAtomicWriteFile_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	notADir := filepath.Join(dir, "nope")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	dest := filepath.Join(notADir, "out.txt")
	if err := AtomicWriteFile(dest, []byte("x"), 0o600, false); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAtomicWriteFile_OverwriteRenameFallbackSuccess(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "target")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("seed dir: %v", err)
	}
	if err := AtomicWriteFile(dest, []byte("ok"), 0o600, true); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("expected file, got dir")
	}
}

func TestAtomicWriteFile_OverwriteRenameFallbackRemoveError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "target")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatalf("seed dir: %v", err)
	}
	// Make directory non-empty so os.Remove fails.
	if err := os.WriteFile(filepath.Join(dest, "child"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed child: %v", err)
	}
	if err := AtomicWriteFile(dest, []byte("ok"), 0o600, true); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAtomicWriteFile_ErrorsViaInjection(t *testing.T) {
	t.Run("StatError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		old := statFn
		statFn = func(string) (os.FileInfo, error) { return nil, errors.New("boom") }
		defer func() { statFn = old }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, false); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("CreateTempError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		old := createTempFn
		createTempFn = func(string, string) (*os.File, error) { return nil, errors.New("boom") }
		defer func() { createTempFn = old }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("WriteTempError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		oldWrite := writeFn
		writeFn = func(*os.File, []byte) (int, error) { return 0, errors.New("boom") }
		defer func() { writeFn = oldWrite }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("CloseTempError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		oldClose := closeFn
		closeFn = func(f *os.File) error {
			_ = f.Close()
			return errors.New("boom")
		}
		defer func() { closeFn = oldClose }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("ChmodTempError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		old := chmodFn
		chmodFn = func(string, os.FileMode) error { return errors.New("boom") }
		defer func() { chmodFn = old }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("RenameErrorOverwriteFalse", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		old := renameFn
		renameFn = func(string, string) error { return errors.New("boom") }
		defer func() { renameFn = old }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, false); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("RenameErrorOverwriteTrue", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		old := renameFn
		renameFn = func(string, string) error { return errors.New("boom") }
		defer func() { renameFn = old }()

		if err := AtomicWriteFile(dest, []byte("x"), 0o600, true); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("RenameFallbackReturnsSecondRenameError", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, "out.txt")

		oldRename := renameFn
		oldRemove := removeFn
		defer func() {
			renameFn = oldRename
			removeFn = oldRemove
		}()

		attempt := 0
		renameFn = func(string, string) error {
			attempt++
			if attempt == 1 {
				return errors.New("first rename failed")
			}
			return errors.New("second rename failed")
		}
		removeFn = func(string) error { return nil }

		err := AtomicWriteFile(dest, []byte("x"), 0o600, true)
		if err == nil {
			t.Fatalf("expected error")
		}
		if !strings.Contains(err.Error(), "second rename failed") {
			t.Fatalf("expected second rename failure in error, got %v", err)
		}
	})
}
