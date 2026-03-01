package secrettype

import (
	"fmt"
	"sort"

	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

const (
	NameOpaque              = "opaque"
	NameCertificate         = "certificate"
	NameKeyValue            = "key_value"
	NameBasicCredentials    = "basic_credentials"
	NameDatabaseCredentials = "database_credentials"
	NameSSHKey              = "ssh_key"
)

var allowed = map[string]struct{}{
	NameOpaque:              {},
	NameCertificate:         {},
	NameKeyValue:            {},
	NameBasicCredentials:    {},
	NameDatabaseCredentials: {},
	NameSSHKey:              {},
}

func IsValid(name string) bool {
	_, ok := allowed[name]
	return ok
}

func Names() []string {
	out := make([]string, 0, len(allowed))
	for k := range allowed {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ToScaleway(name string) (secret.SecretType, error) {
	switch name {
	case NameOpaque:
		return secret.SecretTypeOpaque, nil
	case NameCertificate:
		return secret.SecretTypeCertificate, nil
	case NameKeyValue:
		return secret.SecretTypeKeyValue, nil
	case NameBasicCredentials:
		return secret.SecretTypeBasicCredentials, nil
	case NameDatabaseCredentials:
		return secret.SecretTypeDatabaseCredentials, nil
	case NameSSHKey:
		return secret.SecretTypeSSHKey, nil
	default:
		return "", fmt.Errorf("unsupported secret type mapping for %q", name)
	}
}
