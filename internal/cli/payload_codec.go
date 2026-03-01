package cli

import (
	"encoding/json"
	"fmt"

	"github.com/bsmartlabs/dev-vault/internal/dotenv"
)

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
