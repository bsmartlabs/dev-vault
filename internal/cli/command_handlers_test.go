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

func TestRunHandlersHelpAndParseErrors(t *testing.T) {
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

func TestRunListTableWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
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

func TestRunPullAndRunPushOutputWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"x-pull-dev":{"file":"in.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"x-push-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}
		}
	}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	api := newFakeSecretAPI()
	pullSecret := api.AddSecret("proj", "x-pull-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(pullSecret.ID, []byte("DATA"))
	pushSecret := api.AddSecret("proj", "x-push-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(pushSecret.ID, []byte("DATA"))
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var pullErr bytes.Buffer
	pullCode := runPull(commandContext{
		stdout:     &failingWriter{},
		stderr:     &pullErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"x-pull-dev", "--overwrite"})
	if pullCode != 1 {
		t.Fatalf("expected pull exit 1, got %d stderr=%s", pullCode, pullErr.String())
	}

	var pushErr bytes.Buffer
	pushCode := runPush(commandContext{
		stdout:     &failingWriter{},
		stderr:     &pushErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"x-push-dev", "--description", "d"})
	if pushCode != 1 {
		t.Fatalf("expected push exit 1, got %d stderr=%s", pushCode, pushErr.String())
	}
}

func TestRunPullAndPushReportPartialBatchFailures(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"a-pull-dev":{"file":"a.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"b-pull-dev":{"file":"b.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"a-push-dev":{"file":"a.bin","format":"raw","path":"/","mode":"push","type":"opaque"},
			"b-push-dev":{"file":"b.bin","format":"raw","path":"/","mode":"push","type":"opaque"}
		}
	}`)
	if err := os.WriteFile(filepath.Join(root, "a.bin"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write a.bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.bin"), []byte("B"), 0o644); err != nil {
		t.Fatalf("write b.bin: %v", err)
	}

	api := newFakeSecretAPI()
	aPull := api.AddSecret("proj", "a-pull-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(aPull.ID, []byte("A"))
	aPush := api.AddSecret("proj", "a-push-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(aPush.ID, []byte("A"))
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var pullOut, pullErr bytes.Buffer
	pullCode := runPull(commandContext{
		stdout:     &pullOut,
		stderr:     &pullErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"--all", "--overwrite"})
	if pullCode != 1 {
		t.Fatalf("expected pull exit 1, got %d stderr=%s", pullCode, pullErr.String())
	}
	if got := pullOut.String(); !bytes.Contains([]byte(got), []byte("pulled a-pull-dev")) {
		t.Fatalf("expected pull success line for a-pull-dev, got %q", got)
	}
	if got := pullErr.String(); !bytes.Contains([]byte(got), []byte("failed pull b-pull-dev")) || !bytes.Contains([]byte(got), []byte("pull completed with failures: 1/2 failed")) {
		t.Fatalf("unexpected pull stderr: %q", got)
	}

	var pushOut, pushErr bytes.Buffer
	pushCode := runPush(commandContext{
		stdout:     &pushOut,
		stderr:     &pushErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"--all", "--yes"})
	if pushCode != 1 {
		t.Fatalf("expected push exit 1, got %d stderr=%s", pushCode, pushErr.String())
	}
	if got := pushOut.String(); !bytes.Contains([]byte(got), []byte("pushed a-push-dev")) {
		t.Fatalf("expected push success line for a-push-dev, got %q", got)
	}
	if got := pushErr.String(); !bytes.Contains([]byte(got), []byte("failed push b-push-dev")) || !bytes.Contains([]byte(got), []byte("push completed with failures: 1/2 failed")) {
		t.Fatalf("unexpected push stderr: %q", got)
	}
}

func TestRunPullAndPushSingleFailureUsesBatchErrorContract(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"single-pull-dev":{"file":"single.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"single-push-dev":{"file":"single.bin","format":"raw","path":"/","mode":"push","type":"opaque"}
		}
	}`)
	if err := os.WriteFile(filepath.Join(root, "single.bin"), []byte("S"), 0o644); err != nil {
		t.Fatalf("write single.bin: %v", err)
	}
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return newFakeSecretAPI(), nil })

	var pullOut, pullErr bytes.Buffer
	pullCode := runPull(commandContext{
		stdout:     &pullOut,
		stderr:     &pullErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"single-pull-dev", "--overwrite"})
	if pullCode != 1 {
		t.Fatalf("expected pull exit 1, got %d stderr=%s", pullCode, pullErr.String())
	}
	if got := pullErr.String(); !bytes.Contains([]byte(got), []byte("failed pull single-pull-dev")) {
		t.Fatalf("unexpected pull stderr: %q", got)
	}
	if got := pullErr.String(); !bytes.Contains([]byte(got), []byte("pull completed with failures: 1/1 failed")) {
		t.Fatalf("expected batch summary for single failure, got %q", got)
	}

	var pushOut, pushErr bytes.Buffer
	pushCode := runPush(commandContext{
		stdout:     &pushOut,
		stderr:     &pushErr,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"single-push-dev"})
	if pushCode != 1 {
		t.Fatalf("expected push exit 1, got %d stderr=%s", pushCode, pushErr.String())
	}
	if got := pushErr.String(); !bytes.Contains([]byte(got), []byte("failed push single-push-dev")) {
		t.Fatalf("unexpected push stderr: %q", got)
	}
	if got := pushErr.String(); !bytes.Contains([]byte(got), []byte("push completed with failures: 1/1 failed")) {
		t.Fatalf("expected batch summary for single failure, got %q", got)
	}
}

func TestRunPullAndPushPartialFailureStderrWriteError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"a-pull-dev":{"file":"a.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"b-pull-dev":{"file":"b.bin","format":"raw","path":"/","mode":"pull","type":"opaque"},
			"a-push-dev":{"file":"a.bin","format":"raw","path":"/","mode":"push","type":"opaque"},
			"b-push-dev":{"file":"b.bin","format":"raw","path":"/","mode":"push","type":"opaque"}
		}
	}`)
	if err := os.WriteFile(filepath.Join(root, "a.bin"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write a.bin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.bin"), []byte("B"), 0o644); err != nil {
		t.Fatalf("write b.bin: %v", err)
	}

	api := newFakeSecretAPI()
	aPull := api.AddSecret("proj", "a-pull-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(aPull.ID, []byte("A"))
	aPush := api.AddSecret("proj", "a-push-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(aPush.ID, []byte("A"))
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	pullCode := runPull(commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &failingWriter{},
		configPath: cfgPath,
		deps:       deps,
	}, []string{"--all", "--overwrite"})
	if pullCode != 1 {
		t.Fatalf("expected pull exit 1, got %d", pullCode)
	}

	pushCode := runPush(commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &failingWriter{},
		configPath: cfgPath,
		deps:       deps,
	}, []string{"--all", "--yes"})
	if pushCode != 1 {
		t.Fatalf("expected push exit 1, got %d", pushCode)
	}
}

func TestRunPullAllowsDashPrefixedPositionalAfterSentinel(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"--config-dev":{"file":"dash.bin","format":"raw","path":"/","mode":"pull","type":"opaque"}
		}
	}`)

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "--config-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DASH"))
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	var out, errBuf bytes.Buffer
	code := runPull(commandContext{
		stdout:     &out,
		stderr:     &errBuf,
		configPath: cfgPath,
		deps:       deps,
	}, []string{"--overwrite", "--", "--config-dev"})
	if code != 0 {
		t.Fatalf("expected pull exit 0, got %d stderr=%s", code, errBuf.String())
	}
	if got := out.String(); !bytes.Contains([]byte(got), []byte("pulled --config-dev")) {
		t.Fatalf("unexpected stdout: %q", got)
	}
}
