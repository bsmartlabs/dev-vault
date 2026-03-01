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

func TestParseLongFlagToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantName string
		wantEq   bool
		wantOK   bool
	}{
		{name: "NonLong", token: "-h", wantOK: false},
		{name: "DoubleDashOnly", token: "--", wantOK: false},
		{name: "NoValue", token: "--config", wantName: "config", wantEq: false, wantOK: true},
		{name: "WithValue", token: "--profile=ci", wantName: "profile", wantEq: true, wantOK: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotEq, gotOK := parseLongFlagToken(tt.token)
			if gotName != tt.wantName || gotEq != tt.wantEq || gotOK != tt.wantOK {
				t.Fatalf("parseLongFlagToken(%q) = (%q,%t,%t), want (%q,%t,%t)", tt.token, gotName, gotEq, gotOK, tt.wantName, tt.wantEq, tt.wantOK)
			}
		})
	}
}

func TestIsGlobalOptionFlag(t *testing.T) {
	if takesValue, ok := isGlobalOptionFlag("config"); !ok || !takesValue {
		t.Fatalf("expected config global flag with value, got ok=%t takesValue=%t", ok, takesValue)
	}
	if _, ok := isGlobalOptionFlag("json"); ok {
		t.Fatal("json should not be a global option flag")
	}
}
