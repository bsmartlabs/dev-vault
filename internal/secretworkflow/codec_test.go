package secretworkflow

import (
	"strings"
	"testing"
)

func TestJSONToDotenv(t *testing.T) {
	dotenvPayload, err := JSONToDotenv([]byte(`{"A":"1","B":2}`))
	if err != nil {
		t.Fatalf("JSONToDotenv: %v", err)
	}
	rendered := string(dotenvPayload)
	if !strings.Contains(rendered, `A="1"`) || !strings.Contains(rendered, `B="2"`) {
		t.Fatalf("unexpected dotenv payload: %q", rendered)
	}
}

func TestJSONToDotenv_InvalidPayload(t *testing.T) {
	if _, err := JSONToDotenv([]byte("not-json")); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestDotenvToJSON(t *testing.T) {
	jsonPayload, err := DotenvToJSON([]byte("C=3\n"))
	if err != nil {
		t.Fatalf("DotenvToJSON: %v", err)
	}
	if !strings.Contains(string(jsonPayload), `"C":"3"`) {
		t.Fatalf("unexpected json payload: %s", string(jsonPayload))
	}
}

func TestDotenvToJSON_InvalidPayload(t *testing.T) {
	if _, err := DotenvToJSON([]byte("NOPE")); err == nil {
		t.Fatal("expected error for invalid dotenv payload")
	}
}
