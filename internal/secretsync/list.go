package secretsync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

func (s Service) List(query ListQuery) ([]ListRecord, error) {
	req := secretprovider.ListSecretsInput{}
	if query.Path != "" {
		req.Path = query.Path
	}

	if query.Type != "" {
		req.Type = query.Type
	}

	respSecrets, err := s.api.ListSecrets(req)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	filtered := make([]ListRecord, 0, len(respSecrets))
	for _, secretRecord := range respSecrets {
		if !config.IsDevSecretName(secretRecord.Name) {
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
		filtered = append(filtered, ListRecord{
			ID:   secretRecord.ID,
			Name: secretRecord.Name,
			Path: secretRecord.Path,
			Type: string(secretRecord.Type),
		})
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	return filtered, nil
}
