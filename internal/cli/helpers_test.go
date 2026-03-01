package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestHelpersFile_BasicSmoke(t *testing.T) {
	var flags stringSliceFlag
	if err := flags.Set("one"); err != nil {
		t.Fatalf("set flag one: %v", err)
	}
	if err := flags.Set("two"); err != nil {
		t.Fatalf("set flag two: %v", err)
	}
	if got := flags.String(); got != "one,two" {
		t.Fatalf("unexpected stringSliceFlag.String(): %q", got)
	}

	got := reorderFlags([]string{"name-dev", "--overwrite"}, map[string]bool{"overwrite": false})
	want := []string{"--overwrite", "name-dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reorderFlags mismatch: got %#v want %#v", got, want)
	}

	mapping := map[string]config.MappingEntry{
		"a-dev": {Mode: "pull"},
		"b-dev": {Mode: "both"},
	}
	targets, err := selectMappingTargets(mapping, true, nil, "pull")
	if err != nil {
		t.Fatalf("selectMappingTargets(all): %v", err)
	}
	if !reflect.DeepEqual(targets, []string{"a-dev", "b-dev"}) {
		t.Fatalf("unexpected targets: %#v", targets)
	}

	if _, err := parseSecretType("opaque"); err != nil {
		t.Fatalf("parseSecretType(opaque): %v", err)
	}
	if _, err := parseSecretType("not-a-type"); err == nil {
		t.Fatal("expected parseSecretType to fail for unknown type")
	}
	if got := mustParseSecretType("opaque"); got == "" {
		t.Fatal("expected mustParseSecretType to return non-empty value")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected mustParseSecretType panic for invalid type")
		}
	}()
	_ = mustParseSecretType("still-not-valid")

	dotenvPayload, err := jsonToDotenv([]byte(`{"A":"1","B":2}`))
	if err != nil {
		t.Fatalf("jsonToDotenv: %v", err)
	}
	if !strings.Contains(string(dotenvPayload), `A="1"`) {
		t.Fatalf("expected A entry in dotenv payload, got %q", string(dotenvPayload))
	}

	jsonPayload, err := dotenvToJSON([]byte("C=3\n"))
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
