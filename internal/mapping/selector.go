package mapping

import (
	"errors"
	"fmt"
	"sort"
)

func SelectTargetsForMode(entries map[string]Entry, all bool, positional []string, mode Mode) ([]Target, error) {
	if all && len(positional) > 0 {
		return nil, errors.New("cannot use --all with explicit secret names")
	}
	if !all && len(positional) == 0 {
		return nil, errors.New("no secrets specified (use --all or pass secret names)")
	}
	if !mode.IsSupportedCommandMode() {
		return nil, fmt.Errorf("unsupported command mode: %s", mode)
	}

	if all {
		targets := make([]Target, 0, len(entries))
		for name, entry := range entries {
			if entry.Mode.AllowsCommand(mode) {
				targets = append(targets, Target{Name: name, Entry: entry})
			}
		}
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Name < targets[j].Name
		})
		if len(targets) == 0 {
			return nil, fmt.Errorf("no mapping entries selected for %s", mode)
		}
		return targets, nil
	}

	seen := make(map[string]struct{}, len(positional))
	targets := make([]Target, 0, len(positional))
	for _, name := range positional {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		entry, ok := entries[name]
		if !ok {
			return nil, fmt.Errorf("secret not found in mapping: %s", name)
		}
		if !entry.Mode.AllowsCommand(mode) {
			return nil, fmt.Errorf("mapping entry %s has mode=%s, cannot be used with %s", name, entry.Mode, mode)
		}
		targets = append(targets, Target{Name: name, Entry: entry})
	}

	return targets, nil
}
