package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultConfigName = ".scw.json"

var (
	absFn      = filepath.Abs
	relFn      = filepath.Rel
	statFileFn = os.Stat
	readFileFn = os.ReadFile
)

type MappingEntry struct {
	File   string `json:"file"`
	Format string `json:"format,omitempty"` // raw|dotenv
	Path   string `json:"path,omitempty"`   // default "/"
	Mode   string `json:"mode,omitempty"`   // pull|push|sync default sync
	Type   string `json:"type,omitempty"`   // expected secret type
}

type Config struct {
	OrganizationID string                  `json:"organization_id"`
	ProjectID      string                  `json:"project_id"`
	Region         string                  `json:"region"`
	Profile        string                  `json:"profile,omitempty"`
	Mapping        map[string]MappingEntry `json:"mapping"`
}

type Loaded struct {
	Path string
	Root string
	Cfg  Config
}

func FindConfigPath(startDir string) (string, error) {
	if startDir == "" {
		return "", errors.New("startDir is empty")
	}

	dir, err := absFn(startDir)
	if err != nil {
		return "", fmt.Errorf("abs startDir: %w", err)
	}

	for {
		candidate := filepath.Join(dir, DefaultConfigName)
		if info, err := statFileFn(candidate); err == nil && !info.IsDir() {
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
		found, err := FindConfigPath(startDir)
		if err != nil {
			return nil, err
		}
		path = found
	}

	absPath, err := absFn(path)
	if err != nil {
		return nil, fmt.Errorf("abs config path: %w", err)
	}

	raw, err := readFileFn(absPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config json: %w", err)
	}

	if err := cfg.normalizeAndValidate(); err != nil {
		return nil, err
	}

	root := filepath.Dir(absPath)
	return &Loaded{Path: absPath, Root: root, Cfg: cfg}, nil
}

func (c *Config) normalizeAndValidate() error {
	if strings.TrimSpace(c.OrganizationID) == "" {
		return errors.New("missing required field: organization_id")
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		return errors.New("missing required field: project_id")
	}
	if strings.TrimSpace(c.Region) == "" {
		return errors.New("missing required field: region")
	}
	if c.Mapping == nil {
		return errors.New("missing required field: mapping")
	}
	if len(c.Mapping) == 0 {
		return errors.New("mapping is empty")
	}

	for name, entry := range c.Mapping {
		if !strings.HasSuffix(name, "-dev") {
			return fmt.Errorf("mapping key %q must end with -dev", name)
		}

		entry.File = strings.TrimSpace(entry.File)
		if entry.File == "" {
			return fmt.Errorf("mapping %q: missing required field: file", name)
		}
		if filepath.IsAbs(entry.File) {
			return fmt.Errorf("mapping %q: file must be relative, got %q", name, entry.File)
		}

		if entry.Format == "" {
			entry.Format = "raw"
		}
		switch entry.Format {
		case "raw", "dotenv":
		default:
			return fmt.Errorf("mapping %q: invalid format %q", name, entry.Format)
		}

		if entry.Path == "" {
			entry.Path = "/"
		}
		if !strings.HasPrefix(entry.Path, "/") {
			return fmt.Errorf("mapping %q: path must start with '/', got %q", name, entry.Path)
		}

		if entry.Mode == "" {
			entry.Mode = "sync"
		}
		switch entry.Mode {
		case "pull", "push", "sync":
		default:
			return fmt.Errorf("mapping %q: invalid mode %q", name, entry.Mode)
		}

		entry.Type = strings.TrimSpace(entry.Type)

		c.Mapping[name] = entry
	}

	return nil
}

func ResolveFile(rootDir string, rel string) (string, error) {
	if rootDir == "" {
		return "", errors.New("rootDir is empty")
	}
	if rel == "" {
		return "", errors.New("relative path is empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative: %q", rel)
	}

	absRoot, err := absFn(rootDir)
	if err != nil {
		return "", fmt.Errorf("abs rootDir: %w", err)
	}

	absPath, err := absFn(filepath.Join(absRoot, rel))
	if err != nil {
		return "", fmt.Errorf("abs joined path: %w", err)
	}

	relToRoot, err := relFn(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}

	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project root: %q", rel)
	}

	return absPath, nil
}
