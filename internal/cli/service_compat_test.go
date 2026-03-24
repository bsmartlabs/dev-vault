package cli

import (
	"github.com/bsmartlabs/dev-vault/internal/cli/selection"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
)

func selectMappingTargets(entries map[string]mapping.Entry, all bool, positional []string, mode string) ([]string, error) {
	var typedMode mapping.Mode
	switch mode {
	case "pull":
		typedMode = mapping.ModePull
	case "push":
		typedMode = mapping.ModePush
	default:
		typedMode = mapping.Mode("")
	}
	targets, err := selection.SelectTargetsForMode(entries, all, positional, typedMode)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names, nil
}
