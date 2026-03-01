package cli

import (
	"bytes"
	"errors"
	"flag"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestHasHelpFlag(t *testing.T) {
	if !hasHelpFlag([]string{"-h"}) {
		t.Fatal("expected -h to be recognized")
	}
	if !hasHelpFlag([]string{"--help"}) {
		t.Fatal("expected --help to be recognized")
	}
	if !hasHelpFlag([]string{"-help"}) {
		t.Fatal("expected -help to be recognized")
	}
	if hasHelpFlag([]string{"--", "-h"}) {
		t.Fatal("expected -- sentinel to stop help-flag detection")
	}
}

func TestParseCommand_SingleDashHelpPath(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := commandContext{
		stdout: &out,
		stderr: &errBuf,
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"-help"}, listCommandDef)
	if parsed != nil {
		t.Fatalf("expected nil parsed command on help, got %#v", parsed)
	}
	if exitCodeForError(err) != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCodeForError(err))
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
}

func TestParseCommand_SingleDashHelpWriteFailure(t *testing.T) {
	ctx := commandContext{
		stdout: &failingWriter{},
		stderr: &bytes.Buffer{},
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"-help"}, listCommandDef)
	if parsed != nil {
		t.Fatalf("expected nil parsed command on help failure, got %#v", parsed)
	}
	if exitCodeForError(err) != 1 {
		t.Fatalf("expected output failure code 1, got %d", exitCodeForError(err))
	}
}

func TestFlagValueHelpers(t *testing.T) {
	t.Run("BoolFlagValue", func(t *testing.T) {
		trueValue := true
		falseValue := false
		values := parsedFlagValues{
			boolValues: map[string]*bool{
				"true":  &trueValue,
				"false": &falseValue,
				"nil":   nil,
			},
		}
		if !boolFlagValue(values, "true") {
			t.Fatal("expected true flag value")
		}
		if boolFlagValue(values, "false") {
			t.Fatal("expected false flag value")
		}
		if boolFlagValue(values, "missing") {
			t.Fatal("expected missing bool flag value to be false")
		}
		if boolFlagValue(values, "nil") {
			t.Fatal("expected nil bool flag pointer to be false")
		}
	})

	t.Run("StringFlagValue", func(t *testing.T) {
		text := "value"
		values := parsedFlagValues{
			stringValues: map[string]*string{
				"set": &text,
				"nil": nil,
			},
		}
		if got := stringFlagValue(values, "set"); got != "value" {
			t.Fatalf("unexpected string flag value: %q", got)
		}
		if got := stringFlagValue(values, "missing"); got != "" {
			t.Fatalf("expected empty string for missing value, got %q", got)
		}
		if got := stringFlagValue(values, "nil"); got != "" {
			t.Fatalf("expected empty string for nil pointer, got %q", got)
		}
	})

	t.Run("SliceFlagValue", func(t *testing.T) {
		value := stringSliceFlag{"a", "b"}
		empty := stringSliceFlag{}
		values := parsedFlagValues{
			sliceValues: map[string]*stringSliceFlag{
				"set":   &value,
				"empty": &empty,
				"nil":   nil,
			},
		}
		got := sliceFlagValue(values, "set")
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("unexpected slice flag value: %#v", got)
		}
		got[0] = "changed"
		if value[0] != "a" {
			t.Fatalf("expected returned slice to be copied, got original=%#v", value)
		}
		if got := sliceFlagValue(values, "empty"); got != nil {
			t.Fatalf("expected nil for empty slice value, got %#v", got)
		}
		if got := sliceFlagValue(values, "missing"); got != nil {
			t.Fatalf("expected nil for missing slice value, got %#v", got)
		}
		if got := sliceFlagValue(values, "nil"); got != nil {
			t.Fatalf("expected nil for nil pointer, got %#v", got)
		}
	})
}
