package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func loadAndOpenAPI(configPath, profileOverride string, stderr io.Writer, deps Dependencies) (*config.Loaded, SecretAPI, bool) {
	wd, err := getwdFn()
	if err != nil {
		fmt.Fprintf(stderr, "getwd: %v\n", err)
		return nil, nil, false
	}
	loaded, err := config.Load(wd, configPath)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return nil, nil, false
	}
	api, err := deps.OpenSecretAPI(loaded.Cfg, profileOverride)
	if err != nil {
		fmt.Fprintf(stderr, "open scaleway api: %v\n", err)
		return nil, nil, false
	}
	return loaded, api, true
}

type notFoundError struct {
	name string
	path string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("secret not found: name=%s path=%s", e.name, e.path)
}

func resolveSecretByNameAndPath(api SecretAPI, cfg config.Config, name, path string) (*secret.Secret, error) {
	req := secret.ListSecretsRequest{
		Region:               scw.Region(cfg.Region),
		ProjectID:            scw.StringPtr(cfg.ProjectID),
		Name:                 scw.StringPtr(name),
		Path:                 scw.StringPtr(path),
		ScheduledForDeletion: false,
	}
	respSecrets, err := listSecretsByTypes(api, &req, allSecretTypes)
	if err != nil {
		return nil, err
	}

	matches := make([]*secret.Secret, 0, len(respSecrets))
	for _, s := range respSecrets {
		if s != nil && s.Name == name && s.Path == path {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, &notFoundError{name: name, path: path}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, s := range matches {
			ids = append(ids, s.ID)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("multiple secrets match name=%s path=%s: %s", name, path, strings.Join(ids, ","))
	}
	return matches[0], nil
}

func listSecretsByTypes(api SecretAPI, base *secret.ListSecretsRequest, types []secret.SecretType) ([]*secret.Secret, error) {
	var out []*secret.Secret
	for _, t := range types {
		req := *base
		req.Type = t
		resp, err := api.ListSecrets(&req, scw.WithAllPages())
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Secrets...)
	}
	return out, nil
}
