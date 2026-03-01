package secretcontract

import (
	"fmt"
	"sort"
	"strings"
)

const (
	TypeOpaque        = "opaque"
	TypeCertificate   = "certificate"
	TypeKeyValue      = "key_value"
	TypeBasicCreds    = "basic_credentials"
	TypeDatabaseCreds = "database_credentials"
	TypeSSHKey        = "ssh_key"

	RevisionLatestEnabled = "latest_enabled"
)

func Names() []string {
	out := []string{
		TypeOpaque,
		TypeCertificate,
		TypeKeyValue,
		TypeBasicCreds,
		TypeDatabaseCreds,
		TypeSSHKey,
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
