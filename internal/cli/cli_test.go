package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/pathpolicy"
	testsecretapi "github.com/bsmartlabs/dev-vault/internal/testdouble/secretapi"
)

func newFakeSecretAPI() *fakeSecretAPI {
	return testsecretapi.NewFakeSecretAPI()
}

type fakeSecretAPI = testsecretapi.FakeSecretAPI

func writeConfig(t *testing.T, dir string, cfg string) string {
	t.Helper()
	p := filepath.Join(dir, config.DefaultConfigName)
	if err := os.WriteFile(p, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func baseDeps(open func(cfg config.Config, profileOverride string) (SecretAPI, error)) Dependencies {
	return Dependencies{
		Version:            "v",
		Commit:             "c",
		Date:               "d",
		OpenSecretAPI:      open,
		Now:                func() time.Time { return time.Unix(123, 0) },
		Hostname:           func() (string, error) { return "host", nil },
		ResolveProjectPath: pathpolicy.ResolveProjectFile,
	}
}
