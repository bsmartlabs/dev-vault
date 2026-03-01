package cli

import "testing"

func TestNewCommandCatalog(t *testing.T) {
	valid := commandDef{
		Name:      "ok",
		RunParsed: func(commandContext, *parsedCommand) int { return 0 },
	}
	state := newCommandCatalog(valid)
	if len(state.ordered) != 1 {
		t.Fatalf("expected one command, got %d", len(state.ordered))
	}
	if _, ok := state.byName["ok"]; !ok {
		t.Fatal("expected command to be indexed by name")
	}
}

func TestNewCommandCatalog_PanicsOnInvalidDefs(t *testing.T) {
	assertPanics := func(t *testing.T, name string, run func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s: expected panic", name)
			}
		}()
		run()
	}

	validRun := func(commandContext, *parsedCommand) int { return 0 }

	assertPanics(t, "EmptyName", func() {
		_ = newCommandCatalog(commandDef{
			RunParsed: validRun,
		})
	})

	assertPanics(t, "MissingRunParsed", func() {
		_ = newCommandCatalog(commandDef{
			Name: "missing",
		})
	})

	assertPanics(t, "DuplicateName", func() {
		_ = newCommandCatalog(
			commandDef{Name: "dup", RunParsed: validRun},
			commandDef{Name: "dup", RunParsed: validRun},
		)
	})
}
