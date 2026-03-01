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

type failAfterWriter struct {
	okWrites int
	writes   int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.writes >= w.okWrites {
		return 0, errors.New("write failure")
	}
	w.writes++
	return len(p), nil
}

func TestRun_WriteFailureBranches(t *testing.T) {
	deps := Dependencies{}
	if code := Run([]string{"dev-vault", "list"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected internal dependency error to return 1, got %d", code)
	}

	if code := Run([]string{}, &bytes.Buffer{}, &failingWriter{}, baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})); code != 1 {
		t.Fatalf("expected empty-args usage write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "--help"}, &failingWriter{}, &bytes.Buffer{}, baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})); code != 1 {
		t.Fatalf("expected top-level help write failure to return 1, got %d", code)
	}

	deps = baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})
	if code := Run([]string{"dev-vault", "help", "unknown"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected help unknown write failure to return 1, got %d", code)
	}
	if code := Run([]string{"dev-vault", "unknown"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected unknown command write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "help"}, &failingWriter{}, &bytes.Buffer{}, deps); code != 1 {
		t.Fatalf("expected help usage write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "--profile", "ci"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected missing command usage write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "help", "list"}, &failingWriter{}, &bytes.Buffer{}, deps); code != 1 {
		t.Fatalf("expected command help write failure to return 1, got %d", code)
	}

	stderrFailOnSecondWrite := &failAfterWriter{okWrites: 1}
	if code := Run([]string{"dev-vault", "help", "unknown"}, &bytes.Buffer{}, stderrFailOnSecondWrite, deps); code != 1 {
		t.Fatalf("expected unknown-help follow-up usage write failure to return 1, got %d", code)
	}

	stderrUnknownFailOnSecondWrite := &failAfterWriter{okWrites: 1}
	if code := Run([]string{"dev-vault", "unknown"}, &bytes.Buffer{}, stderrUnknownFailOnSecondWrite, deps); code != 1 {
		t.Fatalf("expected unknown-command follow-up usage write failure to return 1, got %d", code)
	}
}

func TestPrintConfigWarnings_WriteFailureStops(t *testing.T) {
	if err := printConfigWarnings(&failingWriter{}, []string{"one", "two"}); err == nil {
		t.Fatal("expected warning write error")
	}
}

func TestRunVersionParsed_WriteFailure(t *testing.T) {
	code := runVersionParsed(commandContext{
		stdout: &failingWriter{},
		stderr: &bytes.Buffer{},
		deps: Dependencies{
			Version: "v",
			Commit:  "c",
			Date:    "d",
		},
	}, &parsedCommand{})
	if code != 1 {
		t.Fatalf("expected version write failure to return 1, got %d", code)
	}
}

func TestRunList_TableRowWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"sync"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	writer := &failAfterWriter{okWrites: 1}
	var errBuf bytes.Buffer
	code := runList(commandContext{
		stdout:     writer,
		stderr:     &errBuf,
		configPath: cfgPath,
		deps:       deps,
	}, []string{})
	if code != 1 {
		t.Fatalf("expected row write failure exit 1, got %d stderr=%s", code, errBuf.String())
	}
}

func TestRuntimeExecute_ErrorWriteFailureStillReturnsExitCode(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"both","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	api.createVerErr = errors.New("version boom")
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })

	code := runPush(commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &failingWriter{},
		configPath: cfgPath,
		deps:       deps,
	}, []string{"x-dev"})
	if code != 1 {
		t.Fatalf("expected runtime error exit 1, got %d", code)
	}
}

func TestRunList_ConfigWarningWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"sync"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", secret.SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	loaded, _, err := loadAndOpenAPI(cfgPath, "", deps)
	if err != nil {
		t.Fatalf("loadAndOpenAPI: %v", err)
	}
	if len(loaded.Warnings) == 0 {
		t.Fatal("expected at least one config warning")
	}

	code := runList(commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &failingWriter{},
		configPath: cfgPath,
		deps:       deps,
	}, []string{})
	if code != 1 {
		t.Fatalf("expected warning write failure to return 1, got %d", code)
	}
}

func TestRunList_HelpUsageWriteFailure(t *testing.T) {
	code := runList(commandContext{
		stdout: &bytes.Buffer{},
		stderr: &failingWriter{},
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
	}, []string{"-h"})
	if code != 1 {
		t.Fatalf("expected help usage write failure to return 1, got %d", code)
	}
}
