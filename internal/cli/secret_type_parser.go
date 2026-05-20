package cli

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func parseSecretType(s string) (secretprovider.SecretType, error) {
	if !secretcontract.IsType(s) {
		return "", fmt.Errorf("unknown secret type %q", s)
	}
	return secretcontract.Type(s), nil
}
