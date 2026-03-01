package cli

import "testing"

func TestCommandCatalog_Lookups(t *testing.T) {
	defs := commandDefs()
	if len(defs) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(defs))
	}

	names := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if def.Name == "" {
			t.Fatal("command name must not be empty")
		}
		if def.RunParsed == nil {
			t.Fatalf("command %q has nil RunParsed", def.Name)
		}
		if _, exists := names[def.Name]; exists {
			t.Fatalf("duplicate command definition: %q", def.Name)
		}
		names[def.Name] = struct{}{}
	}

	if _, ok := commandForName("pull"); !ok {
		t.Fatal("expected pull command to exist")
	}
	if _, ok := commandForName("missing"); ok {
		t.Fatal("expected missing command lookup to fail")
	}
}
