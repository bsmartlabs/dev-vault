package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies("v1", "c1", "d1", func(cfg config.Config, profileOverride string) (SecretAPI, error) {
		return nil, nil
	})
	if deps.Version != "v1" || deps.Commit != "c1" || deps.Date != "d1" {
		t.Fatalf("unexpected deps: %#v", deps)
	}
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil {
		t.Fatalf("expected all funcs set: %#v", deps)
	}
}

func TestLoadAndOpenAPI_GetwdError(t *testing.T) {
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	})
	deps.Getwd = func() (string, error) { return "", errors.New("boom") }
	_, _, err := loadAndOpenAPI("", "", deps)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPI_Success(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x","mode":"sync"}}}`)

	old, _ := os.Getwd()
	defer func() { _ = os.Chdir(old) }()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	api := newFakeSecretAPI()
	deps := baseDeps(func(cfg config.Config, s string) (SecretAPI, error) { return api, nil })
	loaded, gotAPI, err := loadAndOpenAPI(cfgPath, "", deps)
	if err != nil || loaded == nil || gotAPI == nil {
		t.Fatalf("expected success, got err=%v loaded=%v api=%v", err, loaded, gotAPI)
	}
}

func TestLoadAndOpenAPI_ConfigError(t *testing.T) {
	_, _, err := loadAndOpenAPI("/nope.json", "", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadAndOpenAPI_OpenError(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{"organization_id":"org","project_id":"proj","region":"fr-par","mapping":{"x-dev":{"file":"x"}}}`)
	_, _, err := loadAndOpenAPI(cfgPath, "", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, errors.New("boom")
	}))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRun_ProfileOverridePropagatesToOpenSecretAPI(t *testing.T) {
	root := t.TempDir()
	cfgPath := writeConfig(t, root, `{
  "organization_id":"org",
  "project_id":"proj",
  "region":"fr-par",
  "mapping":{"x-dev":{"file":"x","mode":"sync"}}
}`)

	var gotProfile string
	deps := baseDeps(func(cfg config.Config, profile string) (SecretAPI, error) {
		gotProfile = profile
		return newFakeSecretAPI(), nil
	})

	var out, errBuf bytes.Buffer
	code := Run([]string{"dev-vault", "--config", cfgPath, "--profile", "ci-prof", "list"}, &out, &errBuf, deps)
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, errBuf.String())
	}
	if gotProfile != "ci-prof" {
		t.Fatalf("expected profile override to propagate, got %q", gotProfile)
	}
	if !strings.Contains(out.String(), "NAME") || !strings.Contains(out.String(), "TYPE") {
		t.Fatalf("unexpected list output: %s", out.String())
	}
}

func TestParseCommandErrorContract(t *testing.T) {
	base := errors.New("parse boom")
	parseErr := &parseCommandError{code: 2, err: base}
	if parseErr.Error() != "parse boom" {
		t.Fatalf("unexpected parse error string: %q", parseErr.Error())
	}
	if parseErr.Unwrap() != base {
		t.Fatalf("unexpected unwrap: %v", parseErr.Unwrap())
	}

	fallback := &parseCommandError{code: 0}
	if fallback.Error() != "command parse exit" {
		t.Fatalf("unexpected fallback error: %q", fallback.Error())
	}

	code, terminal := parseCommandExitCode(nil)
	if terminal || code != 0 {
		t.Fatalf("nil parse error should continue, got code=%d terminal=%v", code, terminal)
	}

	code, terminal = parseCommandExitCode(parseErr)
	if !terminal || code != 2 {
		t.Fatalf("parse command error should exit usage, got code=%d terminal=%v", code, terminal)
	}

	code, terminal = parseCommandExitCode(errors.New("other"))
	if !terminal || code != 1 {
		t.Fatalf("non-parse error should map to runtime, got code=%d terminal=%v", code, terminal)
	}
}

func TestOpenScalewaySecretAPIWrapper(t *testing.T) {
	_, err := OpenScalewaySecretAPI(config.Config{
		OrganizationID: "00000000-0000-0000-0000-000000000000",
		ProjectID:      "00000000-0000-0000-0000-000000000000",
		Region:         "bad-region",
	}, "")
	if err == nil {
		t.Fatalf("expected wrapper to propagate open error")
	}
}
