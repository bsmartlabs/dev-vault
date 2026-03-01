package cli

type SecretRecord struct {
	ID        string
	ProjectID string
	Name      string
	Path      string
	Type      string
}

type ListSecretsInput struct {
	Region    string
	ProjectID string
	Name      string
	Path      string
	Type      string
}

type AccessSecretVersionInput struct {
	Region   string
	SecretID string
	Revision string
}

type SecretVersionRecord struct {
	SecretID string
	Revision uint32
	Data     []byte
	Type     string
	Status   string
}

type CreateSecretInput struct {
	Region    string
	ProjectID string
	Name      string
	Path      string
	Type      string
}

type CreateSecretVersionInput struct {
	Region          string
	SecretID        string
	Data            []byte
	Description     *string
	DisablePrevious *bool
}

type SecretLister interface {
	ListSecrets(req ListSecretsInput) ([]*SecretRecord, error)
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
