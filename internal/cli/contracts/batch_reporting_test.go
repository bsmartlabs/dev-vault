package contracts_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/cli"
	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/pathpolicy"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type fakeAPI struct{}

func (fakeAPI) ListSecrets(secretprovider.ListSecretsInput) ([]secretprovider.SecretRecord, error) {
	return nil, nil
}

func (fakeAPI) AccessSecretVersion(secretprovider.AccessSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	return nil, errors.New("not implemented")
}

func (fakeAPI) CreateSecret(secretprovider.CreateSecretInput) (*secretprovider.SecretRecord, error) {
	return nil, errors.New("not implemented")
}

func (fakeAPI) CreateSecretVersion(secretprovider.CreateSecretVersionInput) (*secretprovider.SecretVersionRecord, error) {
	return nil, errors.New("not implemented")
}

func writeProjectConfig(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, config.DefaultConfigName)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func depsWithOpen(openFn func(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error)) cli.Dependencies {
	return cli.Dependencies{
		Version:            "v",
		Commit:             "c",
		Date:               "d",
		OpenSecretAPI:      openFn,
		Now:                time.Now,
		Hostname:           func() (string, error) { return "host", nil },
		ResolveProjectPath: pathpolicy.ResolveProjectFile,
	}
}

func TestPullBoundaryContract_OpenAPIError(t *testing.T) {
	cfgPath := writeProjectConfig(t, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{"x-dev":{"file":"x","mode":"pull"}}
	}`)

	var out, errBuf bytes.Buffer
	code := cli.Run(
		[]string{"dev-vault", "--config", cfgPath, "pull", "x-dev"},
		&out,
		&errBuf,
		depsWithOpen(func(config.Config, string) (secretprovider.SecretAPI, error) {
			return nil, errors.New("open failed")
		}),
	)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "open secret api: open failed") {
		t.Fatalf("expected open secret api error in stderr, got %q", errBuf.String())
	}
}

func TestPullBoundaryContract_ServiceInitError(t *testing.T) {
	cfgPath := writeProjectConfig(t, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{"x-dev":{"file":"x","mode":"pull"}}
	}`)

	var out, errBuf bytes.Buffer
	code := cli.Run(
		[]string{"dev-vault", "--config", cfgPath, "pull", "x-dev"},
		&out,
		&errBuf,
		depsWithOpen(func(config.Config, string) (secretprovider.SecretAPI, error) {
			return nil, nil
		}),
	)
	if code != 1 {
		t.Fatalf("expected runtime exit code 1, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "init secret sync service") {
		t.Fatalf("expected service init error in stderr, got %q", errBuf.String())
	}
}

func TestPushBoundaryContract_PreflightStopsBeforeOpen(t *testing.T) {
	cfgPath := writeProjectConfig(t, `{
		"organization_id":"org",
		"project_id":"proj",
		"region":"fr-par",
		"mapping":{
			"a-dev":{"file":"a","mode":"push"},
			"b-dev":{"file":"b","mode":"push"}
		}
	}`)

	openCalled := false
	deps := depsWithOpen(func(config.Config, string) (secretprovider.SecretAPI, error) {
		openCalled = true
		return fakeAPI{}, nil
	})

	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"dev-vault", "--config", cfgPath, "push", "--all"}, &out, &errBuf, deps)
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
	if openCalled {
		t.Fatal("expected preflight to stop before opening secret API")
	}
}
