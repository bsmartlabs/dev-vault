package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestNewCommandInvocationBuildsContextAndParams(t *testing.T) {
	configPath := "manifest.json"
	profileOverride := "staging"
	args := []string{"secret-dev"}
	var stdout, stderr bytes.Buffer

	ctx, params, err := newCommandInvocation(
		baseDeps(func(config.Config, string) (SecretAPI, error) {
			return newFakeSecretAPI(), nil
		}),
		&stdout,
		&stderr,
		&configPath,
		&profileOverride,
		commandConfigValidated,
		args,
	)
	if err != nil {
		t.Fatalf("newCommandInvocation: %v", err)
	}

	if ctx.stdout != &stdout || ctx.stderr != &stderr {
		t.Fatalf("expected command context to keep output writers")
	}
	if ctx.deps.OpenSecretAPI == nil {
		t.Fatalf("expected command context dependencies")
	}
	if params.configPath != configPath || params.profileOverride != profileOverride {
		t.Fatalf("unexpected command params: %#v", params)
	}
	if params.configPolicy != commandConfigValidated {
		t.Fatalf("unexpected config policy: %v", params.configPolicy)
	}
	if len(params.args) != 1 || params.args[0] != args[0] {
		t.Fatalf("unexpected args: %#v", params.args)
	}
}

func TestNewCommandInvocationRejectsMissingRuntimeDeps(t *testing.T) {
	configPath := "manifest.json"
	profileOverride := ""

	_, _, err := newCommandInvocation(
		Dependencies{},
		&bytes.Buffer{},
		&bytes.Buffer{},
		&configPath,
		&profileOverride,
		commandConfigProjectOnly,
		nil,
	)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	if !strings.Contains(err.Error(), "internal error: missing dependencies") {
		t.Fatalf("unexpected error: %v", err)
	}
}
