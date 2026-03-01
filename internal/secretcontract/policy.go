package secretcontract

import (
	"fmt"
	"sort"
	"strings"
)

const (
	TypeOpaque        Type = "opaque"
	TypeCertificate   Type = "certificate"
	TypeKeyValue      Type = "key_value"
	TypeBasicCreds    Type = "basic_credentials"
	TypeDatabaseCreds Type = "database_credentials"
	TypeSSHKey        Type = "ssh_key"

	RevisionLatestEnabled RevisionSelector = "latest_enabled"
)

type Type string
type RevisionSelector string

func Names() []string {
	out := []string{
		string(TypeOpaque),
		string(TypeCertificate),
		string(TypeKeyValue),
		string(TypeBasicCreds),
		string(TypeDatabaseCreds),
		string(TypeSSHKey),
	}
	sort.Strings(out)
	return out
}

func IsDevSecretName(name string) bool {
	return strings.HasSuffix(name, "-dev")
}

func ValidateDevSecretName(name string) error {
	if IsDevSecretName(name) {
		return nil
	}
	return fmt.Errorf("mapping key %q must end with -dev", name)
}
