package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type commandMode int

const (
	commandModePull commandMode = iota + 1
	commandModePush
)

func (m commandMode) String() string {
	switch m {
	case commandModePull:
		return "pull"
	case commandModePush:
		return "push"
	default:
		return "unknown"
	}
}

func (m commandMode) allows(entry config.MappingEntry) bool {
	switch m {
	case commandModePull:
		return entry.Mode.AllowsPull()
	case commandModePush:
		return entry.Mode.AllowsPush()
	default:
		return false
	}
}

func selectMappingTargetsForMode(mapping map[string]config.MappingEntry, all bool, positional []string, mode commandMode) ([]secretsync.MappingTarget, error) {
	if all && len(positional) > 0 {
		return nil, usageError(errors.New("cannot use --all with explicit secret names"))
	}
	if !all && len(positional) == 0 {
		return nil, usageError(errors.New("no secrets specified (use --all or pass secret names)"))
	}

	if mode != commandModePull && mode != commandModePush {
		return nil, usageError(fmt.Errorf("unsupported command mode: %s", mode.String()))
	}

	if all {
		targets := make([]secretsync.MappingTarget, 0, len(mapping))
		for name, entry := range mapping {
			if mode.allows(entry) {
				targets = append(targets, secretsync.MappingTarget{Name: name, Entry: secretsync.MappingEntryFromConfig(entry)})
			}
		}
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Name < targets[j].Name
		})
		if len(targets) == 0 {
			return nil, usageError(fmt.Errorf("no mapping entries selected for %s", mode.String()))
		}
		return targets, nil
	}

	seen := make(map[string]struct{}, len(positional))
	targets := make([]secretsync.MappingTarget, 0, len(positional))
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
		if !mode.allows(entry) {
			return nil, usageError(fmt.Errorf("secret %s not allowed in %s mode (mapping.mode=%s)", name, mode.String(), entry.Mode))
		}
		targets = append(targets, secretsync.MappingTarget{Name: name, Entry: secretsync.MappingEntryFromConfig(entry)})
	}

	return targets, nil
}
