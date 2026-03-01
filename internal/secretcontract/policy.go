package secretcontract

import "sort"

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
