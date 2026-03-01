package secrettype

import (
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type Name string

const (
	NameOpaque              Name = Name(secretcontract.TypeOpaque)
	NameCertificate         Name = Name(secretcontract.TypeCertificate)
	NameKeyValue            Name = Name(secretcontract.TypeKeyValue)
	NameBasicCredentials    Name = Name(secretcontract.TypeBasicCreds)
	NameDatabaseCredentials Name = Name(secretcontract.TypeDatabaseCreds)
	NameSSHKey              Name = Name(secretcontract.TypeSSHKey)
)

var allowed = map[string]struct{}{
	string(NameOpaque):              {},
	string(NameCertificate):         {},
	string(NameKeyValue):            {},
	string(NameBasicCredentials):    {},
	string(NameDatabaseCredentials): {},
	string(NameSSHKey):              {},
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
