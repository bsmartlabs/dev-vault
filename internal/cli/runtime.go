package cli

import "github.com/bsmartlabs/dev-vault/internal/config"

type commandRuntime struct {
	loaded  *config.Loaded
	api     SecretAPI
	service commandService
}

func buildCommandRuntime(configPath, profileOverride string, deps Dependencies) (*commandRuntime, error) {
	loaded, api, err := loadAndOpenAPI(configPath, profileOverride, deps)
	if err != nil {
		return nil, err
	}
	return &commandRuntime{
		loaded:  loaded,
		api:     api,
		service: newCommandService(loaded, api, deps),
	}, nil
}
