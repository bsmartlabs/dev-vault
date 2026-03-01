package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

func parseSecretType(s string) (secretprovider.SecretType, error) {
	if !secrettype.IsValid(s) {
		return "", fmt.Errorf("unknown secret type %q", s)
	}
	return secretprovider.SecretType(s), nil
}
