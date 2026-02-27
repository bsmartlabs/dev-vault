package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/dotenv"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func reorderFlags(argv []string, takesValue map[string]bool) []string {
	// Go's standard flag package stops parsing when it sees the first non-flag argument.
	// For a better CLI UX, accept flags after positional args by reordering them.
	var flags []string
	var positional []string

	normalize := func(tok string) string {
		tok = strings.TrimLeft(tok, "-")
		if i := strings.IndexByte(tok, '='); i >= 0 {
			tok = tok[:i]
		}
		return tok
	}

	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if tok == "--" {
			positional = append(positional, argv[i+1:]...)
			break
		}
		if strings.HasPrefix(tok, "-") && tok != "-" {
			flags = append(flags, tok)
			name := normalize(tok)
			if takesValue[name] && !strings.Contains(tok, "=") && i+1 < len(argv) {
				flags = append(flags, argv[i+1])
				i++
			}
			continue
		}
		positional = append(positional, tok)
	}

	return append(flags, positional...)
}

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

func jsonToDotenv(payload []byte) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	env := make(map[string]string, len(m))
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			env[k] = vv
		default:
			// Values come from json.Unmarshal into interface{}, so they are always JSON-marshalable.
			env[k] = string(mustJSONMarshal(v))
		}
	}
	return dotenv.Render(env), nil
}

func mustJSONMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func dotenvToJSON(payload []byte) ([]byte, error) {
	env, err := dotenv.Parse(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(env)
}
