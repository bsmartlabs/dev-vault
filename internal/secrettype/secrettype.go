package secrettype

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

var allowed = map[string]struct{}{
	string(secretcontract.TypeOpaque):        {},
	string(secretcontract.TypeCertificate):   {},
	string(secretcontract.TypeKeyValue):      {},
	string(secretcontract.TypeBasicCreds):    {},
	string(secretcontract.TypeDatabaseCreds): {},
	string(secretcontract.TypeSSHKey):        {},
}

func IsValid(name string) bool {
	_, ok := allowed[name]
	return ok
}

func Names() []string {
	return secretcontract.Names()
}

func ToScaleway(name string) (secret.SecretType, error) {
	switch secretcontract.Type(name) {
	case secretcontract.TypeOpaque:
		return secret.SecretTypeOpaque, nil
	case secretcontract.TypeCertificate:
		return secret.SecretTypeCertificate, nil
	case secretcontract.TypeKeyValue:
		return secret.SecretTypeKeyValue, nil
	case secretcontract.TypeBasicCreds:
		return secret.SecretTypeBasicCredentials, nil
	case secretcontract.TypeDatabaseCreds:
		return secret.SecretTypeDatabaseCredentials, nil
	case secretcontract.TypeSSHKey:
		return secret.SecretTypeSSHKey, nil
	default:
		return "", fmt.Errorf("unsupported secret type mapping for %q", name)
	}
}
