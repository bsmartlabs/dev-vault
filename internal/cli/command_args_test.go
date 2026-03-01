package cli

import (
	"flag"
	"strings"
	"testing"
)

func TestRejectUnexpectedArgs(t *testing.T) {
	t.Run("NilParsed", func(t *testing.T) {
		if err := rejectUnexpectedArgs(nil, "list"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("NoExtraArgs", func(t *testing.T) {
		parsed := &parsedCommand{fs: flag.NewFlagSet("list", flag.ContinueOnError)}
		if err := parsed.fs.Parse(nil); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if err := rejectUnexpectedArgs(parsed, "list"); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("ExtraArgs", func(t *testing.T) {
		parsed := &parsedCommand{fs: flag.NewFlagSet("list", flag.ContinueOnError)}
		if err := parsed.fs.Parse([]string{"one", "two"}); err != nil {
			t.Fatalf("parse: %v", err)
		}

		err := rejectUnexpectedArgs(parsed, "list")
		if err == nil {
			t.Fatal("expected error")
		}
		if code := exitCodeForError(err); code != 2 {
			t.Fatalf("expected usage exit code 2, got %d", code)
		}
		if !strings.Contains(err.Error(), "list does not accept positional arguments: one two") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
