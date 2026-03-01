package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func (s commandService) list(query listQuery) ([]listRecord, error) {
	req := secretprovider.ListSecretsInput{
		Region:    s.cfg.Region,
		ProjectID: s.cfg.ProjectID,
	}
	if query.Path != "" {
		req.Path = query.Path
	}

	secretTypes := supportedSecretTypes()
	if query.Type != "" {
		st, err := parseSecretType(query.Type)
		if err != nil {
			return nil, usageError(fmt.Errorf("invalid --type: %w", err))
		}
		secretTypes = []secretprovider.SecretType{st}
	}

	respSecrets, err := listSecretsByTypes(s.api, req, secretTypes)
	if err != nil {
		return nil, runtimeError(err)
	}

	filtered := make([]listRecord, 0, len(respSecrets))
	for _, secretRecord := range respSecrets {
		if !config.IsDevSecretName(secretRecord.Name) {
			continue
		}
		if query.Path != "" && secretRecord.Path != query.Path {
			continue
		}
		if len(query.NameContains) > 0 {
			miss := false
			for _, c := range query.NameContains {
				if !strings.Contains(secretRecord.Name, c) {
					miss = true
					break
				}
			}
			if miss {
				continue
			}
		}
		if query.NameRegex != nil && !query.NameRegex.MatchString(secretRecord.Name) {
			continue
		}
		filtered = append(filtered, listRecord{
			ID:   secretRecord.ID,
			Name: secretRecord.Name,
			Path: secretRecord.Path,
			Type: string(secretRecord.Type),
		})
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered, nil
}
