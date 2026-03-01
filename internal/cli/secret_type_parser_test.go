package cli

import (
	"strings"
	"testing"
)

func TestParseSecretType(t *testing.T) {
	got, err := parseSecretType("opaque")
	if err != nil {
		t.Fatalf("parseSecretType(opaque): %v", err)
	}
	if got != "opaque" {
		t.Fatalf("unexpected parsed type: %q", got)
	}

	_, err = parseSecretType("invalid-type")
	if err == nil {
		t.Fatal("expected invalid type error")
	}
	if !strings.Contains(err.Error(), "unknown secret type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
