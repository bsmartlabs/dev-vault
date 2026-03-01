package cli

import (
	"strings"
	"testing"
)

func TestPayloadCodecModule_Smoke(t *testing.T) {
	dotenvPayload, err := jsonToDotenv([]byte(`{"A":"1","B":2}`))
	if err != nil {
		t.Fatalf("jsonToDotenv: %v", err)
	}
	if !strings.Contains(string(dotenvPayload), `A="1"`) {
		t.Fatalf("missing expected dotenv key: %q", string(dotenvPayload))
	}
	jsonPayload, err := dotenvToJSON([]byte("C=3\n"))
	if err != nil {
		t.Fatalf("dotenvToJSON: %v", err)
	}
	if !strings.Contains(string(jsonPayload), `"C":"3"`) {
		t.Fatalf("unexpected json payload: %s", string(jsonPayload))
	}
}
