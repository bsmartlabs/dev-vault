package cli

import (
	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	scwprovider "github.com/bsmartlabs/dev-vault/internal/secretprovider/scaleway"
)

type SecretType = secretprovider.SecretType

const (
	SecretTypeOpaque      = secretprovider.SecretTypeOpaque
	SecretTypeCertificate = secretprovider.SecretTypeCertificate
	SecretTypeKeyValue    = secretprovider.SecretTypeKeyValue
)

type RevisionSelector = secretprovider.RevisionSelector

const RevisionLatestEnabled = secretprovider.RevisionLatestEnabled

type SecretRecord = secretprovider.SecretRecord
type ListSecretsInput = secretprovider.ListSecretsInput
type AccessSecretVersionInput = secretprovider.AccessSecretVersionInput
type SecretVersionRecord = secretprovider.SecretVersionRecord
type CreateSecretInput = secretprovider.CreateSecretInput
type CreateSecretVersionInput = secretprovider.CreateSecretVersionInput

type SecretLister = secretprovider.SecretLister
type SecretVersionAccessor = secretprovider.SecretVersionAccessor
type SecretCreator = secretprovider.SecretCreator
type SecretVersionCreator = secretprovider.SecretVersionCreator
type SecretAPI = secretprovider.SecretAPI

func OpenScalewaySecretAPI(cfg config.Config, profileOverride string) (SecretAPI, error) {
	return scwprovider.Open(cfg, profileOverride)
}
