package main

import (
	"io"
	"os"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/cli"
)

func TestRunMain_UsesInjectedRunnerAndBuildMetadata(t *testing.T) {
	got := runMain([]string{"dev-vault"}, io.Discard, io.Discard, "v", "c", "d", func(args []string, stdout, stderr io.Writer, deps cli.Dependencies) int {
		if deps.Version != "v" || deps.Commit != "c" || deps.Date != "d" {
			t.Fatalf("unexpected deps: %#v", deps)
		}
		if len(args) != 1 || args[0] != "dev-vault" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return 42
	})
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestMain_UsesExitFnWithRunMainStatus(t *testing.T) {
	origArgs := os.Args
	origExit := exitFn
	t.Cleanup(func() {
		os.Args = origArgs
		exitFn = origExit
	})

	os.Args = []string{"dev-vault", "-h"}
	exitCode := -1
	exitFn = func(code int) {
		exitCode = code
	}

	main()

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}
