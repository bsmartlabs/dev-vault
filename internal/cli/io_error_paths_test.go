package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
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

func TestRunWriteFailureBranches(t *testing.T) {
	deps := Dependencies{}
	if code := Run([]string{"dev-vault", "list"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected internal dependency error to return 1, got %d", code)
	}

	// Empty args with failing stderr: root RunE fires, writes usage to stderr (fails),
	// then returns usageError. The error print to stderr also fails, so exit 1.
	if code := Run([]string{}, &bytes.Buffer{}, &failingWriter{}, baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})); code != 1 {
		t.Fatalf("expected empty-args stderr write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "--help"}, &failingWriter{}, &bytes.Buffer{}, baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})); code != 1 {
		t.Fatalf("expected top-level help write failure to return 1, got %d", code)
	}
	if code := Run([]string{"dev-vault", "-help"}, &failingWriter{}, &bytes.Buffer{}, baseDeps(func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return newFakeSecretAPI(), nil
	})); code != 1 {
		t.Fatalf("expected top-level -help write failure to return 1, got %d", code)
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

	// "help" with no args but failing stdout → writeUsage fails → outputError.
	if code := Run([]string{"dev-vault", "help"}, &failingWriter{}, &bytes.Buffer{}, deps); code != 1 {
		t.Fatalf("expected help usage write failure to return 1, got %d", code)
	}

	// "help list" but failing stdout → writeUsage for subcommand fails → outputError.
	if code := Run([]string{"dev-vault", "help", "list"}, &failingWriter{}, &bytes.Buffer{}, deps); code != 1 {
		t.Fatalf("expected command help write failure to return 1, got %d", code)
	}

	// "help -help" → the -help flag is stripped, treated as bare "help" → write to failingWriter.
	if code := Run([]string{"dev-vault", "help", "-help"}, &failingWriter{}, &bytes.Buffer{}, deps); code != 1 {
		t.Fatalf("expected help -help write failure to return 1, got %d", code)
	}

	if code := Run([]string{"dev-vault", "--profile", "ci"}, &bytes.Buffer{}, &failingWriter{}, deps); code != 1 {
		t.Fatalf("expected missing command usage write failure to return 1, got %d", code)
	}
}

func TestRunVersionWriteFailure(t *testing.T) {
	code := Run([]string{"dev-vault", "version"}, &failingWriter{}, &bytes.Buffer{}, Dependencies{
		Version: "v",
		Commit:  "c",
		Date:    "d",
	})
	if code != 1 {
		t.Fatalf("expected version write failure to return 1, got %d", code)
	}
}

func TestRunListTableRowWriteFailure(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", SecretTypeOpaque)
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

func TestRuntimeExecuteErrorWriteFailureStillReturnsExitCode(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"in.bin","format":"raw","path":"/","mode":"push","type":"opaque"}}}`)
	if err := os.WriteFile(filepath.Join(root, "in.bin"), []byte("DATA"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	api := newFakeSecretAPI()
	sec := api.AddSecret("proj", "x-dev", "/", SecretTypeOpaque)
	api.AddEnabledVersion(sec.ID, []byte("DATA"))
	api.CreateVerErr = errors.New("version boom")
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

func TestRunListModeCleanupLeavesListPathClean(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"pull"}}}`)
	api := newFakeSecretAPI()
	api.AddSecret("proj", "x-dev", "/", SecretTypeOpaque)
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	code := runList(commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &failingWriter{},
		configPath: cfgPath,
		deps:       deps,
	}, []string{})
	if code != 0 {
		t.Fatalf("expected success with no warning writes, got %d", code)
	}
}

func TestRunListHelpUsageWriteFailure(t *testing.T) {
	code := runList(commandContext{
		stdout: &failingWriter{},
		stderr: &bytes.Buffer{},
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
	}, []string{"-h"})
	if code != 1 {
		t.Fatalf("expected help usage write failure to return 1, got %d", code)
	}
}
