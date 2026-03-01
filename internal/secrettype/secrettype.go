package secrettype

import (
	"fmt"
	"sort"

	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

var allowed = map[string]struct{}{
	"opaque":               {},
	"certificate":          {},
	"key_value":            {},
	"basic_credentials":    {},
	"database_credentials": {},
	"ssh_key":              {},
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
	case "opaque":
		return secret.SecretTypeOpaque, nil
	case "certificate":
		return secret.SecretTypeCertificate, nil
	case "key_value":
		return secret.SecretTypeKeyValue, nil
	case "basic_credentials":
		return secret.SecretTypeBasicCredentials, nil
	case "database_credentials":
		return secret.SecretTypeDatabaseCredentials, nil
	case "ssh_key":
		return secret.SecretTypeSSHKey, nil
	default:
		return "", fmt.Errorf("unsupported secret type mapping for %q", name)
	}
}
