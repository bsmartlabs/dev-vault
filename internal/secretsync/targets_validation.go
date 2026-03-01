package secretsync

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
)

func validateBatchTargets(targets []MappingTarget, expectedMode mapping.Mode) error {
	for _, target := range targets {
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
	}
	return nil
}
