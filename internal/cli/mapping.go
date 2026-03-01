package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

func selectMappingTargets(mapping map[string]config.MappingEntry, all bool, positional []string, mode string) ([]string, error) {
	if all && len(positional) > 0 {
		return nil, usageError(errors.New("cannot use --all with explicit secret names"))
	}
	if !all && len(positional) == 0 {
		return nil, usageError(errors.New("no secrets specified (use --all or pass secret names)"))
	}

	isAllowedMode := func(entry config.MappingEntry) bool {
		switch mode {
		case "pull":
			return entry.Mode == "pull" || entry.Mode == "both"
		case "push":
			return entry.Mode == "push" || entry.Mode == "both"
		default:
			return false
		}
	}

	var out []string
	if all {
		for name, entry := range mapping {
			if isAllowedMode(entry) {
				out = append(out, name)
			}
		}
		sort.Strings(out)
		if len(out) == 0 {
			return nil, usageError(fmt.Errorf("no mapping entries selected for %s", mode))
		}
		return out, nil
	}

	seen := make(map[string]struct{}, len(positional))
	for _, name := range positional {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		if !config.IsDevSecretName(name) {
			return nil, usageError(fmt.Errorf("refusing non-dev secret name: %s", name))
		}
		entry, ok := mapping[name]
		if !ok {
			return nil, usageError(fmt.Errorf("secret not found in mapping: %s", name))
		}
		if !isAllowedMode(entry) {
			return nil, usageError(fmt.Errorf("secret %s not allowed in %s mode (mapping.mode=%s)", name, mode, entry.Mode))
		}
		out = append(out, name)
	}
	return out, nil
}

type mappingTarget struct {
	Name  string
	Entry config.MappingEntry
}

func selectMappingCommandTargets(mapping map[string]config.MappingEntry, all bool, positional []string, mode string) ([]mappingTarget, error) {
	names, err := selectMappingTargets(mapping, all, positional, mode)
	if err != nil {
		return nil, err
	}
	targets := make([]mappingTarget, 0, len(names))
	for _, name := range names {
		targets = append(targets, mappingTarget{
			Name:  name,
			Entry: mapping[name],
		})
	}
	return targets, nil
}

func parseSecretType(s string) (secretprovider.SecretType, error) {
	if !secrettype.IsValid(s) {
		return "", fmt.Errorf("unknown secret type %q", s)
	}
	return secretprovider.SecretType(s), nil
}
