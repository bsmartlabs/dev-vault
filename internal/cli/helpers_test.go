package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretworkflow"
)

func jsonToDotenvForTest(payload []byte) ([]byte, error) {
	return secretworkflow.JSONToDotenv(payload)
}

func dotenvToJSONForTest(payload []byte) ([]byte, error) {
	return secretworkflow.DotenvToJSON(payload)
}

func TestHelpersFileBasicSmoke(t *testing.T) {
	mapping := map[string]mapping.Entry{
		"a-dev": {Mode: "pull"},
		"b-dev": {Mode: "pull"},
	}
	targets, err := selectMappingTargets(mapping, true, nil, "pull")
	if err != nil {
		t.Fatalf("selectMappingTargets(all): %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("unexpected targets count: %d", len(targets))
	}

	if _, err := parseSecretType("opaque"); err != nil {
		t.Fatalf("parseSecretType(opaque): %v", err)
	}
	if _, err := parseSecretType("not-a-type"); err == nil {
		t.Fatal("expected parseSecretType to fail for unknown type")
	}

	dotenvPayload, err := jsonToDotenvForTest([]byte(`{"A":"1","B":"2"}`))
	if err != nil {
		t.Fatalf("jsonToDotenv: %v", err)
	}
	if !strings.Contains(string(dotenvPayload), `A="1"`) {
		t.Fatalf("expected A entry in dotenv payload, got %q", string(dotenvPayload))
	}

	jsonPayload, err := dotenvToJSONForTest([]byte("C=3\n"))
	if err != nil {
		t.Fatalf("dotenvToJSON: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(jsonPayload, &m); err != nil {
		t.Fatalf("unmarshal dotenvToJSON payload: %v", err)
	}
	if m["C"] != "3" {
		t.Fatalf("expected C=3 in converted payload, got %#v", m)
	}
}
