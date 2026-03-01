package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

const DefaultConfigName = ".scw.json"

var (
	defaultConfigDeps = configDeps{
		abs:      filepath.Abs,
		rel:      filepath.Rel,
		statFile: os.Stat,
		readFile: os.ReadFile,
	}
)

type configDeps struct {
	abs      func(string) (string, error)
	rel      func(string, string) (string, error)
	statFile func(string) (os.FileInfo, error)
	readFile func(string) ([]byte, error)
}

type MappingFormat string

const (
	MappingFormatRaw    MappingFormat = "raw"
	MappingFormatDotenv MappingFormat = "dotenv"
)

type MappingMode string

const (
	MappingModePull   MappingMode = "pull"
	MappingModePush   MappingMode = "push"
	MappingModeBoth   MappingMode = "both"
	MappingModeLegacy MappingMode = "sync"
)

func (m MappingMode) AllowsPull() bool {
	return m == MappingModePull || m == MappingModeBoth
}

func (m MappingMode) AllowsPush() bool {
	return m == MappingModePush || m == MappingModeBoth
}

type MappingEntry struct {
	File   string        `json:"file"`
	Format MappingFormat `json:"format,omitempty"` // raw|dotenv
	Path   string        `json:"path,omitempty"`   // default "/"
	Mode   MappingMode   `json:"mode,omitempty"`   // pull|push|both (default: both). "sync" is accepted as legacy alias for "both".
	Type   string        `json:"type,omitempty"`   // expected secret type
}

type Config struct {
	OrganizationID string                  `json:"organization_id"`
	ProjectID      string                  `json:"project_id"`
	Region         string                  `json:"region"`
	Profile        string                  `json:"profile,omitempty"`
	Mapping        map[string]MappingEntry `json:"mapping"`
}

type Loaded struct {
	Path     string
	Root     string
	Cfg      Config
	Warnings []string
}

func IsDevSecretName(name string) bool {
	return strings.HasSuffix(name, "-dev")
}

func ValidateDevSecretName(name string) error {
	if IsDevSecretName(name) {
		return nil
	}
	return fmt.Errorf("mapping key %q must end with -dev", name)
}

func FindConfigPath(startDir string) (string, error) {
	return findConfigPath(startDir, defaultConfigDeps)
}

func findConfigPath(startDir string, deps configDeps) (string, error) {
	if startDir == "" {
		return "", errors.New("startDir is empty")
	}

	dir, err := deps.abs(startDir)
	if err != nil {
		return "", fmt.Errorf("abs startDir: %w", err)
	}

	for {
		candidate := filepath.Join(dir, DefaultConfigName)
		if info, err := deps.statFile(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("%s not found from %s upward", DefaultConfigName, startDir)
}

func Load(startDir, explicitPath string) (*Loaded, error) {
	return loadWithDeps(startDir, explicitPath, defaultConfigDeps)
}

func loadWithDeps(startDir, explicitPath string, deps configDeps) (*Loaded, error) {
	if startDir == "" {
		return nil, errors.New("startDir is empty")
	}

	var path string
	if explicitPath != "" {
		if filepath.IsAbs(explicitPath) {
			path = explicitPath
		} else {
			path = filepath.Join(startDir, explicitPath)
		}
	} else {
		found, err := findConfigPath(startDir, deps)
		if err != nil {
			return nil, err
		}
		path = found
	}

	absPath, err := deps.abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs config path: %w", err)
	}

	raw, err := deps.readFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config json: %w", err)
	}
	// Reject trailing JSON tokens after the single top-level config object.
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("decode config json: trailing data after top-level JSON object")
		}
		return nil, fmt.Errorf("decode config json: trailing data after top-level JSON object: %w", err)
	}

	warnings, err := cfg.normalizeAndValidate()
	if err != nil {
		return nil, err
	}

	root := filepath.Dir(absPath)
	return &Loaded{Path: absPath, Root: root, Cfg: cfg, Warnings: warnings}, nil
}

func (c *Config) normalizeAndValidate() ([]string, error) {
	warnings := []string{}

	if strings.TrimSpace(c.OrganizationID) == "" {
		return nil, errors.New("missing required field: organization_id")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return nil, errors.New("missing required field: project_id")
	}
	if strings.TrimSpace(c.Region) == "" {
		return nil, errors.New("missing required field: region")
	}
	if c.Mapping == nil {
		return nil, errors.New("missing required field: mapping")
	}
	if len(c.Mapping) == 0 {
		return nil, errors.New("mapping is empty")
	}

	for name, entry := range c.Mapping {
		if err := ValidateDevSecretName(name); err != nil {
			return nil, err
		}

		entry.File = strings.TrimSpace(entry.File)
		if entry.File == "" {
			return nil, fmt.Errorf("mapping %q: missing required field: file", name)
		}
		if filepath.IsAbs(entry.File) {
			return nil, fmt.Errorf("mapping %q: file must be relative, got %q", name, entry.File)
		}

		if entry.Format == "" {
			entry.Format = MappingFormatRaw
		}
		switch entry.Format {
		case MappingFormatRaw, MappingFormatDotenv:
		default:
			return nil, fmt.Errorf("mapping %q: invalid format %q", name, entry.Format)
		}

		if entry.Path == "" {
			entry.Path = "/"
		}
		if !strings.HasPrefix(entry.Path, "/") {
			return nil, fmt.Errorf("mapping %q: path must start with '/', got %q", name, entry.Path)
		}

		if entry.Mode == "" {
			entry.Mode = MappingModeBoth
		}
		if entry.Mode == MappingModeLegacy {
			// Back-compat: older manifests used "sync" to mean "both".
			warnings = append(warnings, fmt.Sprintf("mapping %q uses legacy mode=sync; use mode=both (sync will be removed in a future major release)", name))
			entry.Mode = MappingModeBoth
		}
		switch entry.Mode {
		case MappingModePull, MappingModePush, MappingModeBoth:
		default:
			return nil, fmt.Errorf("mapping %q: invalid mode %q", name, entry.Mode)
		}

		entry.Type = strings.TrimSpace(entry.Type)
		if entry.Type != "" {
			if !secrettype.IsValid(entry.Type) {
				return nil, fmt.Errorf("mapping %q: invalid type %q", name, entry.Type)
			}
		}

		c.Mapping[name] = entry
	}

	return warnings, nil
}

func ResolveFile(rootDir string, rel string) (string, error) {
	return resolveFileWithDeps(rootDir, rel, defaultConfigDeps)
}

func resolveFileWithDeps(rootDir string, rel string, deps configDeps) (string, error) {
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
