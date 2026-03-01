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
