package secretsync

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
)

func splitTargetsByPolicy(targets []MappingTarget, expectedMode mapping.Mode) ([]MappingTarget, []BatchFailure) {
	valid := make([]MappingTarget, 0, len(targets))
	invalid := make([]BatchFailure, 0, len(targets))
	for _, target := range targets {
		if err := validateTargetPolicy(target, expectedMode); err != nil {
			invalid = append(invalid, BatchFailure{Name: target.Name, Err: err})
			continue
		}
		valid = append(valid, target)
	}
	return valid, invalid
}

func validateTargetPolicy(target MappingTarget, expectedMode mapping.Mode) error {
	if !secretcontract.IsDevSecretName(target.Name) {
		return fmt.Errorf("refusing non-dev secret name: %s", target.Name)
	}
	if target.Entry.Mode != expectedMode {
		return fmt.Errorf(
			"secret %s not allowed in %s mode (mapping.mode=%s)",
			target.Name,
			expectedMode,
			target.Entry.Mode,
		)
	}
	return nil
}
