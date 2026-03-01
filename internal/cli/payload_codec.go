package cli

import (
	"encoding/json"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/dotenv"
)

func jsonToDotenv(payload []byte) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	env := make(map[string]string, len(m))
	for k, raw := range m {
		var asString string
		if err := json.Unmarshal(raw, &asString); err == nil {
			env[k] = asString
			continue
		}
		env[k] = string(raw)
	}
	return dotenv.Render(env), nil
}

func dotenvToJSON(payload []byte) ([]byte, error) {
	env, err := dotenv.Parse(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(env)
}
