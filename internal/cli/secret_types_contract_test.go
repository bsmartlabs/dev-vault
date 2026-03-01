package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
)

func TestSecretTypesContract_CanonicalPolicy(t *testing.T) {
	canonical := secrettype.Names()
	if len(canonical) == 0 {
		t.Fatal("expected canonical secret type policy to be non-empty")
	}

	allowed := make(map[string]struct{}, len(canonical))
	for _, name := range canonical {
		allowed[name] = struct{}{}

		parsed, err := parseSecretType(name)
		if err != nil {
			t.Fatalf("parseSecretType(%q): %v", name, err)
		}
		if parsed == "" {
			t.Fatalf("parseSecretType(%q): empty value", name)
		}
	}

	rejected := []string{"", "opaque ", "OPAQUE", "not-a-secret-type"}
	for _, token := range rejected {
		if _, err := parseSecretType(token); err == nil {
			t.Fatalf("expected parseSecretType(%q) to fail", token)
		}
	}

	runtimeTypes := []secretprovider.SecretType{
		secretprovider.SecretTypeOpaque,
		secretprovider.SecretTypeCertificate,
		secretprovider.SecretTypeKeyValue,
		secretprovider.SecretTypeBasicCredentials,
		secretprovider.SecretTypeDatabaseCredentials,
		secretprovider.SecretTypeSSHKey,
	}
	if len(runtimeTypes) != len(canonical) {
		t.Fatalf("runtime supported types size mismatch: got %d want %d", len(runtimeTypes), len(canonical))
	}
	for _, runtimeType := range runtimeTypes {
		if _, ok := allowed[string(runtimeType)]; !ok {
			t.Fatalf("runtime includes unknown type %q", runtimeType)
		}
	}
}

func TestSecretTypesContract_ListUsageIncludesCanonicalTypes(t *testing.T) {
	var buf bytes.Buffer
	printListUsage(&buf)
	usage := buf.String()

	for _, name := range secrettype.Names() {
		if !strings.Contains(usage, name) {
			t.Fatalf("list usage missing secret type %q", name)
		}
	}
}
