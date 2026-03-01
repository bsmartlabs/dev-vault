package cli

import (
	"flag"
	"reflect"
	"testing"
)

func TestFlagsModule_Smoke(t *testing.T) {
	takes := withGlobalFlagSpecs(map[string]bool{"json": false})
	if !takes["config"] || !takes["profile"] {
		t.Fatalf("expected global keys in spec: %#v", takes)
	}
	fs := flag.NewFlagSet("x", flag.ContinueOnError)
	cfg := ""
	prof := ""
	bindGlobalOptionFlags(fs, &cfg, &prof)
	if err := fs.Parse([]string{"--config", "c", "--profile", "p"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg != "c" || prof != "p" {
		t.Fatalf("unexpected parsed globals: config=%q profile=%q", cfg, prof)
	}

	got := reorderFlags([]string{"name-dev", "--json"}, map[string]bool{"json": false})
	want := []string{"--json", "name-dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reorder mismatch: got %#v want %#v", got, want)
	}
}
