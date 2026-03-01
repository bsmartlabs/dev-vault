package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

func TestRunPullBatch_OpenAPIError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)

	var errBuf bytes.Buffer
	ctx := commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &errBuf,
		configPath: cfgPath,
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, errors.New("open failed")
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}

	code := runPullBatch(ctx, parsed, pullOptions{})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "open secret api: open failed") {
		t.Fatalf("expected open secret api error in stderr, got %q", errBuf.String())
	}
}

func TestRunPullBatch_ServiceInitError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)

	var errBuf bytes.Buffer
	ctx := commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &errBuf,
		configPath: cfgPath,
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return nil, nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}

	code := runPullBatch(ctx, parsed, pullOptions{})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", errBuf.String())
	}
}

func TestRunPushBatch_PreflightStopsBeforeServiceInit(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{
    "a-dev":{"file":"a","mode":"push"},
    "b-dev":{"file":"b","mode":"push"}
  }
}`)

	openCalled := false
	ctx := commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
		configPath: cfgPath,
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			openCalled = true
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"--all"}, pushCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}

	code := runPushBatch(ctx, parsed, pushOptions{all: true})
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
	if openCalled {
		t.Fatal("expected preflight to stop before opening secret API")
	}
}

func TestReportBatchResults(t *testing.T) {
	t.Run("PullReporterWritesSuccessAndFailure", func(t *testing.T) {
		var out, errBuf bytes.Buffer
		err := reportPullBatchResults(
			commandContext{stdout: &out, stderr: &errBuf},
			secretsync.PullBatchResult{
				Succeeded: []secretsync.PullResult{{Name: "a-dev", File: "a.env", Revision: 3, Type: "opaque"}},
				Failed:    []secretsync.BatchFailure{{Name: "b-dev", Err: errors.New("boom")}},
				Summary:   secretsync.BatchSummary{Operation: "pull", Failed: 1, Total: 2},
			},
		)
		var batchErr *secretsync.BatchOperationError
		if !errors.As(err, &batchErr) {
			t.Fatalf("expected batch operation error, got %v", err)
		}
		if !strings.Contains(out.String(), "pulled a-dev -> a.env (rev=3 type=opaque)") {
			t.Fatalf("unexpected stdout: %q", out.String())
		}
		if !strings.Contains(errBuf.String(), "failed pull b-dev: boom") {
			t.Fatalf("unexpected stderr: %q", errBuf.String())
		}
	})

	t.Run("PushReporterReturnsSummaryError", func(t *testing.T) {
		err := reportPushBatchResults(
			commandContext{stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}},
			secretsync.PushBatchResult{
				Summary: secretsync.BatchSummary{Operation: "push", Failed: 1, Total: 2},
			},
		)
		var batchErr *secretsync.BatchOperationError
		if !errors.As(err, &batchErr) {
			t.Fatalf("expected batch operation error, got %v", err)
		}
	})

	t.Run("PullReporterOutputWriteError", func(t *testing.T) {
		err := reportPullBatchResults(
			commandContext{stdout: &failingWriter{}, stderr: &bytes.Buffer{}},
			secretsync.PullBatchResult{
				Succeeded: []secretsync.PullResult{{Name: "a-dev", File: "a.env", Revision: 1, Type: "opaque"}},
			},
		)
		var outputErr *commandError
		if !errors.As(err, &outputErr) {
			t.Fatalf("expected output error, got %v", err)
		}
		if outputErr.kind != commandErrorOutput {
			t.Fatalf("expected commandErrorOutput, got %v", outputErr.kind)
		}
	})
}
