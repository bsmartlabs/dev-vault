package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestRunPullBatch_ServiceInitError(t *testing.T) {
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
		deps: baseDeps(func(config.Config, string) (SecretAPI, error) {
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
	if got := ctx.stderr.(*bytes.Buffer).String(); !strings.Contains(got, "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", got)
	}
}

func TestRunPushBatch_ServiceInitError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"push"}}
}`)

	ctx := commandContext{
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
		configPath: cfgPath,
		deps: baseDeps(func(config.Config, string) (SecretAPI, error) {
			return nil, nil
		}),
	}

	parsed, err := parseCommand(ctx, []string{"x-dev"}, pushCommandDef)
	if err != nil {
		t.Fatalf("parseCommand: %v", err)
	}

	code := runPushBatch(ctx, parsed, pushOptions{})
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if got := ctx.stderr.(*bytes.Buffer).String(); !strings.Contains(got, "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", got)
	}
}
