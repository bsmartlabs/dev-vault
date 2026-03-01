package fsx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrExists = errors.New("file exists")

type fsDeps struct {
	mkdirAll   func(string, os.FileMode) error
	stat       func(string) (os.FileInfo, error)
	createTemp func(string, string) (*os.File, error)
	chmod      func(string, os.FileMode) error
	rename     func(string, string) error
	remove     func(string) error
	write      func(*os.File, []byte) (int, error)
	close      func(*os.File) error
}

func defaultFSDeps() fsDeps {
	return fsDeps{
		mkdirAll:   os.MkdirAll,
		stat:       os.Stat,
		createTemp: os.CreateTemp,
		chmod:      os.Chmod,
		rename:     os.Rename,
		remove:     os.Remove,
		write:      func(f *os.File, data []byte) (int, error) { return f.Write(data) },
		close:      func(f *os.File) error { return f.Close() },
	}
}

func AtomicWriteFile(path string, data []byte, perm os.FileMode, overwrite bool) error {
	return atomicWriteFileWithDeps(path, data, perm, overwrite, defaultFSDeps())
}

func atomicWriteFileWithDeps(path string, data []byte, perm os.FileMode, overwrite bool, deps fsDeps) error {
	dir := filepath.Dir(path)
	if err := deps.mkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", dir, err)
	}

	if !overwrite {
		if _, err := deps.stat(path); err == nil {
			return ErrExists
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", path, err)
		}
	}

	base := filepath.Base(path)
	f, err := deps.createTemp(dir, base+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tmpName := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = deps.remove(tmpName)
		}
	}()

	if _, err := deps.write(f, data); err != nil {
		_ = deps.close(f)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := deps.close(f); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := deps.chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}

	renameErr := deps.rename(tmpName, path)
	if renameErr == nil {
		cleanup = false
		return nil
	}

	if overwrite {
		if rmErr := deps.remove(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return fmt.Errorf("remove existing: %w", rmErr)
		}
		if retryErr := deps.rename(tmpName, path); retryErr == nil {
			cleanup = false
			return nil
		} else {
			return fmt.Errorf("rename temp to dest after overwrite (first attempt: %v): %w", renameErr, retryErr)
		}
	}

	return fmt.Errorf("rename temp to dest: %w", renameErr)
}
