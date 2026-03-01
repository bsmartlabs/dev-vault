package cli

import (
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secrettype"
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

	for _, name := range secrettype.Names() {
		if _, err := parseSecretType(name); err != nil {
			t.Fatalf("expected parseSecretType to accept canonical type %q: %v", name, err)
		}
	}
}

func TestSelectMappingTargets_DedupesExplicitTargetsPreservingOrder(t *testing.T) {
	mapping := map[string]config.MappingEntry{
		"a-dev": {Mode: "both"},
		"b-dev": {Mode: "both"},
	}

	got, err := selectMappingTargets(mapping, false, []string{"a-dev", "b-dev", "a-dev", "b-dev"}, "pull")
	if err != nil {
		t.Fatalf("selectMappingTargets: %v", err)
	}
	want := []string{"a-dev", "b-dev"}
	if len(got) != len(want) {
		t.Fatalf("unexpected target count: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected target at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestCommandModeHelpers(t *testing.T) {
	entry := config.MappingEntry{Mode: "both"}
	if commandModePull.String() != "pull" {
		t.Fatalf("unexpected pull mode string: %q", commandModePull.String())
	}
	if commandModePush.String() != "push" {
		t.Fatalf("unexpected push mode string: %q", commandModePush.String())
	}
	if commandMode(0).String() != "unknown" {
		t.Fatalf("unexpected unknown mode string: %q", commandMode(0).String())
	}
	if !commandModePull.allows(entry) {
		t.Fatalf("pull mode should allow mapping mode both")
	}
	if !commandModePush.allows(entry) {
		t.Fatalf("push mode should allow mapping mode both")
	}
	if commandMode(0).allows(entry) {
		t.Fatalf("unknown mode should not allow mapping entries")
	}
}
