package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
)

func TestCommandErrorHelpers(t *testing.T) {
	base := errors.New("boom")

	if wrapCommandError(commandErrorRuntime, nil) != nil {
		t.Fatal("expected nil to stay nil")
	}

	wrapped := usageError(base)
	var ce *commandError
	if !errors.As(wrapped, &ce) {
		t.Fatalf("expected commandError, got %T", wrapped)
	}
	if ce.Unwrap() != base {
		t.Fatalf("unexpected unwrap result: %#v", ce.Unwrap())
	}

	if got := wrapCommandError(commandErrorRuntime, wrapped); got != wrapped {
		t.Fatalf("expected already-wrapped error to be returned as-is")
	}

	if code := exitCodeForError(nil); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if code := exitCodeForError(wrapped); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if code := exitCodeForError(runtimeError(base)); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if code := exitCodeForError(base); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
}

func TestRunHandlers_HelpAndParseErrors(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})
	commands := []struct {
		name string
		run  func(commandContext, []string) int
	}{
		{name: "list", run: runList},
		{name: "pull", run: runPull},
		{name: "push", run: runPush},
	}

	for _, cmd := range commands {
		t.Run(cmd.name+"_help", func(t *testing.T) {
			var out, errBuf bytes.Buffer
			code := cmd.run(commandContext{
				stdout: &out,
				stderr: &errBuf,
				deps:   deps,
			}, []string{"-h"})
			if code != 0 {
				t.Fatalf("expected 0, got %d", code)
			}
		})

		t.Run(cmd.name+"_parse_error", func(t *testing.T) {
			var out, errBuf bytes.Buffer
			code := cmd.run(commandContext{
				stdout: &out,
				stderr: &errBuf,
				deps:   deps,
			}, []string{"--nope"})
			if code != 2 {
				t.Fatalf("expected 2, got %d", code)
			}
		})
	}
}

func TestRunList_TableWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var errBuf bytes.Buffer
	code := runList(commandContext{
		stdout:     &failingWriter{},
		stderr:     &errBuf,
		configPath: cfgPath,
		deps:       deps,
	}, []string{})
	if code != 1 {
		t.Fatalf("expected 1, got %d stderr=%s", code, errBuf.String())
	}
}

func TestRunPull_And_RunPush_OutputWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"both","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var pullErr bytes.Buffer
	pullCode := runPull(commandContext{
		stdout:     &failingWriter{},
		stderr:     &pullErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"x-dev", "--overwrite"})
	if pullCode != 1 {
		t.Fatalf("expected pull exit 1, got %d stderr=%s", pullCode, pullErr.String())
	}

	var pushErr bytes.Buffer
	pushCode := runPush(commandContext{
		stdout:     &failingWriter{},
		stderr:     &pushErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"x-dev", "--description", "d"})
	if pushCode != 1 {
		t.Fatalf("expected push exit 1, got %d stderr=%s", pushCode, pushErr.String())
	}
}
