package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

func selectMappingTargets(mapping map[string]config.MappingEntry, all bool, positional []string, mode string) ([]string, error) {
	if all && len(positional) > 0 {
		return nil, errors.New("cannot use --all with explicit secret names")
	}
	if !all && len(positional) == 0 {
		return nil, errors.New("no secrets specified (use --all or pass secret names)")
	}

	isAllowedMode := func(entry config.MappingEntry) bool {
		switch mode {
		case "pull":
			return entry.Mode == "pull" || entry.Mode == "both"
		case "push":
			return entry.Mode == "push" || entry.Mode == "both"
		default:
			return false
		}
	}

	var out []string
	if all {
		for name, entry := range mapping {
			if isAllowedMode(entry) {
				out = append(out, name)
			}
		}
		sort.Strings(out)
		if len(out) == 0 {
			return nil, fmt.Errorf("no mapping entries selected for %s", mode)
		}
		return out, nil
	}

	for _, name := range positional {
		if !strings.HasSuffix(name, "-dev") {
			return nil, fmt.Errorf("refusing non-dev secret name: %s", name)
		}
		entry, ok := mapping[name]
		if !ok {
			return nil, fmt.Errorf("secret not found in mapping: %s", name)
		}
		if !isAllowedMode(entry) {
			return nil, fmt.Errorf("secret %s not allowed in %s mode (mapping.mode=%s)", name, mode, entry.Mode)
		}
		out = append(out, name)
	}
	return out, nil
}

func parseSecretType(s string) (secret.SecretType, error) {
	switch s {
	case "opaque":
		return secret.SecretTypeOpaque, nil
	case "certificate":
		return secret.SecretTypeCertificate, nil
	case "key_value":
		return secret.SecretTypeKeyValue, nil
	case "basic_credentials":
		return secret.SecretTypeBasicCredentials, nil
	case "database_credentials":
		return secret.SecretTypeDatabaseCredentials, nil
	case "ssh_key":
		return secret.SecretTypeSSHKey, nil
	default:
		return "", fmt.Errorf("unknown secret type %q", s)
	}
}

func mustParseSecretType(s string) secret.SecretType {
	st, err := parseSecretType(s)
	if err != nil {
		panic(fmt.Sprintf("invalid secret type %q after config validation", s))
	}
	return st
}
