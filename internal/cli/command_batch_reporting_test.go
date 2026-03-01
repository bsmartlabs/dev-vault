package cli

import (
	"bytes"
	"errors"
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
		run: func(service secretsync.Service, targets []secretsync.MappingTarget, opts pullOptions) (batchRunResult[secretsync.PullResult], error) {
			return batchRunResult[secretsync.PullResult]{}, runtimeError(errors.New("boom"))
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
