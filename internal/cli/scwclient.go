package cli

import (
	"fmt"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func OpenScalewaySecretAPI(cfg config.Config, profileOverride string) (SecretAPI, error) {
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

	return secret.NewAPI(client), nil
}
