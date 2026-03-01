package cli

import (
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestMappingModule_Smoke(t *testing.T) {
	mapping := map[string]config.MappingEntry{"a-dev": {Mode: "both"}}
	targets, err := selectMappingTargets(mapping, true, nil, "pull")
	if err != nil {
		t.Fatalf("select all: %v", err)
	}
	if len(targets) != 1 || targets[0] != "a-dev" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
	if _, err := parseSecretType("opaque"); err != nil {
		t.Fatalf("parseSecretType opaque: %v", err)
	}
	if mustParseSecretType("opaque") == "" {
		t.Fatal("expected non-empty secret type")
	}
}
