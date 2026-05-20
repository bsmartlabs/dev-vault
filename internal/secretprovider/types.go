package secretprovider

import "github.com/bsmartlabs/dev-vault/internal/secretcontract"

type SecretType = secretcontract.Type

const (
	SecretTypeOpaque              SecretType = secretcontract.TypeOpaque
	SecretTypeCertificate         SecretType = secretcontract.TypeCertificate
	SecretTypeKeyValue            SecretType = secretcontract.TypeKeyValue
	SecretTypeBasicCredentials    SecretType = secretcontract.TypeBasicCreds
	SecretTypeDatabaseCredentials SecretType = secretcontract.TypeDatabaseCreds
	SecretTypeSSHKey              SecretType = secretcontract.TypeSSHKey
)

type RevisionSelector = secretcontract.RevisionSelector

const RevisionLatestEnabled RevisionSelector = secretcontract.RevisionLatestEnabled

type SecretRecord struct {
	ID        string
	ProjectID string
	Name      string
	Path      string
	Type      SecretType
}

type ListSecretsInput struct {
	Name string
	Path string
	Type SecretType
}

type AccessSecretVersionInput struct {
	SecretID string
	Revision RevisionSelector
}

type SecretVersionRecord struct {
	SecretID string
	Revision uint32
	Data     []byte
	Type     SecretType
	Status   string
}

type CreateSecretInput struct {
	Name string
	Path string
	Type SecretType
}

type CreateSecretVersionInput struct {
	SecretID        string
	Data            []byte
	Description     *string
	DisablePrevious *bool
}

type SecretAPI interface {
	ListSecrets(req ListSecretsInput) ([]SecretRecord, error)
	AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error)
	CreateSecret(req CreateSecretInput) (*SecretRecord, error)
	CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error)
}
