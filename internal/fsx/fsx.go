package fsx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrExists = errors.New("file exists")

var (
	mkdirAllFn   = os.MkdirAll
	statFn       = os.Stat
	createTempFn = os.CreateTemp
	chmodFn      = os.Chmod
	renameFn     = os.Rename
	removeFn     = os.Remove

	writeFn = func(f *os.File, data []byte) (int, error) { return f.Write(data) }
	closeFn = func(f *os.File) error { return f.Close() }
)

func AtomicWriteFile(path string, data []byte, perm os.FileMode, overwrite bool) error {
	dir := filepath.Dir(path)
	if err := mkdirAllFn(dir, 0o755); err != nil {
		return fmt.Errorf("mkdirall %s: %w", dir, err)
	}

	if !overwrite {
		if _, err := statFn(path); err == nil {
			return ErrExists
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", path, err)
		}
	}

	base := filepath.Base(path)
	f, err := createTempFn(dir, base+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tmpName := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := writeFn(f, data); err != nil {
		_ = closeFn(f)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := closeFn(f); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := chmodFn(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}

	renameErr := renameFn(tmpName, path)
	if renameErr == nil {
		cleanup = false
		return nil
	}

	if overwrite {
		if rmErr := removeFn(path); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			return fmt.Errorf("remove existing: %w", rmErr)
		}
		if retryErr := renameFn(tmpName, path); retryErr == nil {
			cleanup = false
			return nil
		} else {
			return fmt.Errorf("rename temp to dest after overwrite (first attempt: %v): %w", renameErr, retryErr)
		}
	}

	return fmt.Errorf("rename temp to dest: %w", renameErr)
}
