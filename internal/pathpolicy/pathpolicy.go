package pathpolicy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type resolverDeps struct {
	abs func(string) (string, error)
	rel func(string, string) (string, error)
}

var defaultResolverDeps = resolverDeps{
	abs: filepath.Abs,
	rel: filepath.Rel,
}

func ResolveProjectFile(rootDir string, rel string) (string, error) {
	return resolveProjectFileWithDeps(rootDir, rel, defaultResolverDeps)
}

func resolveProjectFileWithDeps(rootDir string, rel string, deps resolverDeps) (string, error) {
	if rootDir == "" {
		return "", errors.New("rootDir is empty")
	}
	if rel == "" {
		return "", errors.New("relative path is empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %q", rel)
	}

	absRoot, err := deps.abs(rootDir)
	if err != nil {
		return "", fmt.Errorf("abs rootDir: %w", err)
	}

	absPath, err := deps.abs(filepath.Join(absRoot, rel))
	if err != nil {
		return "", fmt.Errorf("abs joined path: %w", err)
	}

	relToRoot, err := deps.rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root: %q", rel)
	}

	return absPath, nil
}
