package secretprovider

import "github.com/bsmartlabs/dev-vault/internal/secrettype"

type SecretType string

const (
	SecretTypeOpaque              SecretType = secrettype.NameOpaque
	SecretTypeCertificate         SecretType = secrettype.NameCertificate
	SecretTypeKeyValue            SecretType = secrettype.NameKeyValue
	SecretTypeBasicCredentials    SecretType = secrettype.NameBasicCredentials
	SecretTypeDatabaseCredentials SecretType = secrettype.NameDatabaseCredentials
	SecretTypeSSHKey              SecretType = secrettype.NameSSHKey
)

type SecretRevision string

const SecretRevisionLatestEnabled SecretRevision = "latest_enabled"

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
	Revision SecretRevision
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
