package secretprovider

import "github.com/bsmartlabs/dev-vault/internal/secretcontract"

type SecretType string

const (
	SecretTypeOpaque              SecretType = secretcontract.TypeOpaque
	SecretTypeCertificate         SecretType = secretcontract.TypeCertificate
	SecretTypeKeyValue            SecretType = secretcontract.TypeKeyValue
	SecretTypeBasicCredentials    SecretType = secretcontract.TypeBasicCreds
	SecretTypeDatabaseCredentials SecretType = secretcontract.TypeDatabaseCreds
	SecretTypeSSHKey              SecretType = secretcontract.TypeSSHKey
)

type RevisionSelector string

const RevisionLatestEnabled RevisionSelector = secretcontract.RevisionLatestEnabled

type SecretRecord struct {
	ID        string
	ProjectID string
	Name      string
	Path      string
	Type      SecretType
}

type ListSecretsInput struct {
	Region    string
	ProjectID string
	Name      string
	Path      string
	Type      SecretType
}

type AccessSecretVersionInput struct {
	Region   string
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
	Region    string
	ProjectID string
	Name      string
	Path      string
	Type      SecretType
}

type CreateSecretVersionInput struct {
	Region          string
	SecretID        string
	Data            []byte
	Description     *string
	DisablePrevious *bool
}

type SecretLister interface {
	ListSecrets(req ListSecretsInput) ([]SecretRecord, error)
}

type SecretVersionAccessor interface {
	AccessSecretVersion(req AccessSecretVersionInput) (*SecretVersionRecord, error)
}

type SecretCreator interface {
	CreateSecret(req CreateSecretInput) (*SecretRecord, error)
}

type SecretVersionCreator interface {
	CreateSecretVersion(req CreateSecretVersionInput) (*SecretVersionRecord, error)
}

type SecretAPI interface {
	SecretLister
	SecretVersionAccessor
	SecretCreator
	SecretVersionCreator
}
