package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestRunPullBatchServiceInitError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"pull"}}
}`)

	deps := baseDeps(func(config.Config, string) (SecretAPI, error) {
		return nil, nil
	})

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "pull", "x-dev"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", errBuf.String())
	}
}

func TestRunPushBatchServiceInitError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"push"}}
}`)

	deps := baseDeps(func(config.Config, string) (SecretAPI, error) {
		return nil, nil
	})

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "push", "x-dev"}, &out, &errBuf, deps)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", errBuf.String())
	}
}
