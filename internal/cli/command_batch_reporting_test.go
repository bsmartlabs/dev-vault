package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

func TestRunMappingBatchOperation_RunError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)

	ctx := commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
		configPath: cfgPath,
		deps: baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"--all"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, true, opts, mappingBatchOperation[secretsync.PullResult, pullOptions]{
		mode: mapping.ModePull,
		run: func(service secretsync.Service, targets []secretsync.MappingTarget, opts pullOptions) (secretsync.BatchResult[secretsync.PullResult], error) {
			return secretsync.BatchResult[secretsync.PullResult]{}, runtimeError(errors.New("boom"))
		},
		callbacks: batchReportCallbacks[secretsync.PullResult]{
			SuccessLine: func(item secretsync.PullResult) string { return "ok" },
			FailureLine: func(failure secretsync.BatchFailure) string { return "fail" },
		},
	})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
}

func TestRunMappingBatchOperation_PreflightStopsBeforeServiceInit(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
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

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, mappingBatchOperation[secretsync.PullResult, pullOptions]{
		mode: mapping.ModePull,
		preflight: func(opts pullOptions, targets []secretsync.MappingTarget) error {
			return usageError(errors.New("stop-before-open"))
		},
		run: func(service secretsync.Service, targets []secretsync.MappingTarget, opts pullOptions) (secretsync.BatchResult[secretsync.PullResult], error) {
			t.Fatal("run should not be called when preflight fails")
			return secretsync.BatchResult[secretsync.PullResult]{}, nil
		},
		callbacks: batchReportCallbacks[secretsync.PullResult]{
			SuccessLine: func(item secretsync.PullResult) string { return "ok" },
			FailureLine: func(failure secretsync.BatchFailure) string { return "fail" },
		},
	})
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
	if openCalled {
		t.Fatal("expected preflight to stop before opening secret API")
	}
}

func TestRunMappingBatchOperation_ServiceInitError(t *testing.T) {
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
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, pullBatchOperation)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", errBuf.String())
	}
}

func TestRunMappingBatchOperation_OpenAPIError(t *testing.T) {
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
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, pullBatchOperation)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "open secret api: open failed") {
		t.Fatalf("expected open secret api error in stderr, got %q", errBuf.String())
	}
}

func TestRunMappingBatchOperation_InvalidCallbacks(t *testing.T) {
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
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, mappingBatchOperation[secretsync.PullResult, pullOptions]{
		mode: mapping.ModePull,
		run:  nil,
	})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "batch operation run callback is required") {
		t.Fatalf("unexpected stderr: %q", errBuf.String())
	}
}

func TestRunMappingBatchOperation_InvalidSuccessCallback(t *testing.T) {
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
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, mappingBatchOperation[secretsync.PullResult, pullOptions]{
		mode: mapping.ModePull,
		run: func(service secretsync.Service, targets []secretsync.MappingTarget, opts pullOptions) (secretsync.BatchResult[secretsync.PullResult], error) {
			return secretsync.BatchResult[secretsync.PullResult]{}, nil
		},
		callbacks: batchReportCallbacks[secretsync.PullResult]{
			FailureLine: func(failure secretsync.BatchFailure) string { return "fail" },
		},
	})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "batch operation success callback is required") {
		t.Fatalf("unexpected stderr: %q", errBuf.String())
	}
}

func TestRunMappingBatchOperation_InvalidFailureCallback(t *testing.T) {
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
			return newFakeSecretAPI(), nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pullCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}
	opts := parsePullOptions(parsed)

	code := runMappingBatchOperation(ctx, parsed, false, opts, mappingBatchOperation[secretsync.PullResult, pullOptions]{
		mode: mapping.ModePull,
		run: func(service secretsync.Service, targets []secretsync.MappingTarget, opts pullOptions) (secretsync.BatchResult[secretsync.PullResult], error) {
			return secretsync.BatchResult[secretsync.PullResult]{}, nil
		},
		callbacks: batchReportCallbacks[secretsync.PullResult]{
			SuccessLine: func(item secretsync.PullResult) string { return "ok" },
		},
	})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "batch operation failure callback is required") {
		t.Fatalf("unexpected stderr: %q", errBuf.String())
	}
}
