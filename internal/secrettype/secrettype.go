package secrettype

import "github.com/bsmartlabs/dev-vault/internal/secretcontract"

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
