package scaleway

import (
	"fmt"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func Open(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error) {
	profileName := strings.TrimSpace(profileOverride)
	if profileName == "" {
		profileName = strings.TrimSpace(cfg.Profile)
	}

	region, err := scw.ParseRegion(cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("invalid region %q: %w", cfg.Region, err)
	}

	// Keep precedence explicit: env defaults first, profile override last.
	opts := []scw.ClientOption{scw.WithEnv()}
	if profileName != "" {
		scwCfg, err := scw.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("load scaleway config: %w", err)
		}
		prof, err := scwCfg.GetProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("get scaleway profile %q: %w", profileName, err)
		}
		opts = append(opts, scw.WithProfile(prof))
	}

	opts = append(opts,
		scw.WithDefaultOrganizationID(cfg.OrganizationID),
		scw.WithDefaultProjectID(cfg.ProjectID),
		scw.WithDefaultRegion(region),
	)

	client, err := scw.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create scaleway client: %w", err)
	}

	return &API{api: secret.NewAPI(client)}, nil
}

type API struct {
	api scalewaySecretSDK
}

type scalewaySecretSDK interface {
	ListSecrets(req *secret.ListSecretsRequest, opts ...scw.RequestOption) (*secret.ListSecretsResponse, error)
	AccessSecretVersion(req *secret.AccessSecretVersionRequest, opts ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error)
	CreateSecret(req *secret.CreateSecretRequest, opts ...scw.RequestOption) (*secret.Secret, error)
	CreateSecretVersion(req *secret.CreateSecretVersionRequest, opts ...scw.RequestOption) (*secret.SecretVersion, error)
}

func (s *API) ListSecrets(req secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	region, err := scw.ParseRegion(req.Region)
	if err != nil {
		return nil, fmt.Errorf("parse region %q: %w", req.Region, err)
	}

	listReq := &secret.ListSecretsRequest{
		Region:               region,
		ProjectID:            scw.StringPtr(req.ProjectID),
		ScheduledForDeletion: false,
	}
	if req.Type != "" {
		secretType, err := toScalewaySecretType(req.Type)
		if err != nil {
			return nil, err
		}
		listReq.Type = secretType
	}
	if req.Name != "" {
		listReq.Name = scw.StringPtr(req.Name)
	}
	if req.Path != "" {
		listReq.Path = scw.StringPtr(req.Path)
	}

	resp, err := s.api.ListSecrets(listReq, scw.WithAllPages())
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	out := make([]secretprovider.SecretRecord, 0, len(resp.Secrets))
	for _, item := range resp.Secrets {
		if item == nil {
			continue
		}
		out = append(out, secretprovider.SecretRecord{
			ID:        item.ID,
			ProjectID: item.ProjectID,
			Name:      item.Name,
			Path:      item.Path,
			Type:      secretprovider.SecretType(item.Type),
		})
	}
	return out, nil
}

func (s *API) AccessSecretVersion(req secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	region, err := scw.ParseRegion(req.Region)
	if err != nil {
		return nil, fmt.Errorf("parse region %q: %w", req.Region, err)
	}
	resp, err := s.api.AccessSecretVersion(&secret.AccessSecretVersionRequest{
		Region:   region,
		SecretID: req.SecretID,
		Revision: string(req.Revision),
	})
	if err != nil {
		return nil, fmt.Errorf("access secret version: %w", err)
	}
	return &secretprovider.SecretVersionRecord{
		SecretID: resp.SecretID,
		Revision: resp.Revision,
		Data:     resp.Data,
		Type:     secretprovider.SecretType(resp.Type),
	}, nil
}

func (s *API) CreateSecret(req secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	region, err := scw.ParseRegion(req.Region)
	if err != nil {
		return nil, fmt.Errorf("parse region %q: %w", req.Region, err)
	}
	secretType, err := toScalewaySecretType(req.Type)
	if err != nil {
		return nil, err
	}
	path := req.Path
	if path == "" {
		path = "/"
	}

	resp, err := s.api.CreateSecret(&secret.CreateSecretRequest{
		Region:      region,
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Tags:        []string{},
		Description: nil,
		Type:        secretType,
		Path:        scw.StringPtr(path),
		Protected:   false,
		KeyID:       nil,
	})
	if err != nil {
		return nil, fmt.Errorf("create secret: %w", err)
	}
	return &secretprovider.SecretRecord{
		ID:        resp.ID,
		ProjectID: resp.ProjectID,
		Name:      resp.Name,
		Path:      resp.Path,
		Type:      secretprovider.SecretType(resp.Type),
	}, nil
}

func (s *API) CreateSecretVersion(req secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	region, err := scw.ParseRegion(req.Region)
	if err != nil {
		return nil, fmt.Errorf("parse region %q: %w", req.Region, err)
	}
	resp, err := s.api.CreateSecretVersion(&secret.CreateSecretVersionRequest{
		Region:          region,
		SecretID:        req.SecretID,
		Data:            req.Data,
		Description:     req.Description,
		DisablePrevious: req.DisablePrevious,
	})
	if err != nil {
		return nil, fmt.Errorf("create secret version: %w", err)
	}
	return &secretprovider.SecretVersionRecord{
		SecretID: resp.SecretID,
		Revision: resp.Revision,
		Status:   string(resp.Status),
	}, nil
}

func toScalewaySecretType(name secretprovider.SecretType) (secret.SecretType, error) {
	return secrettype.ToScaleway(string(name))
}
