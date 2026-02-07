package main

import (
	"io"
	"os"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/cli"
)

func TestMain_ExitsWithRunCode(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	oldRun, oldExit := run, osExit
	defer func() {
		version, commit, date = oldVersion, oldCommit, oldDate
		run, osExit = oldRun, oldExit
	}()

	version, commit, date = "v", "c", "d"

	var gotExit int
	osExit = func(code int) { gotExit = code }

	run = func(args []string, stdout, stderr io.Writer, deps cli.Dependencies) int {
		if deps.Version != "v" || deps.Commit != "c" || deps.Date != "d" {
			t.Fatalf("unexpected deps: %#v", deps)
		}
		_, _ = stdout.Write([]byte("ok"))
		_, _ = stderr.Write([]byte("err"))
		return 42
	}

	// Ensure args is non-empty for parity with real invocation.
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"dev-vault"}

	main()
	if gotExit != 42 {
		t.Fatalf("expected exit 42, got %d", gotExit)
	}
}
