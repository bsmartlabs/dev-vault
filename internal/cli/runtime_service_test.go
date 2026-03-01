package cli

import (
	"bytes"
	"errors"
	"flag"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestCommandRuntime_NewServiceErrors(t *testing.T) {
	loaded := &config.Loaded{
		Root: ".",
		Cfg: config.Config{
			OrganizationID: "org",
			ProjectID:      "proj",
			Region:         "fr-par",
		},
	}

	newParsed := func() *parsedCommand {
		return &parsedCommand{fs: flag.NewFlagSet("test", flag.ContinueOnError)}
	}

	baseDeps := func(openFn func(config.Config, string) (SecretAPI, error)) Dependencies {
		return Dependencies{
			OpenSecretAPI:      openFn,
			Now:                time.Now,
			Hostname:           func() (string, error) { return "host", nil },
			ResolveProjectPath: func(_, _ string) (string, error) { return "", nil },
		}
	}

	t.Run("OpenAPIError", func(t *testing.T) {
		runtime := newCommandRuntime(commandContext{
			stdout: &bytes.Buffer{},
			stderr: &bytes.Buffer{},
			deps: baseDeps(func(config.Config, string) (SecretAPI, error) {
				return nil, errors.New("open failed")
			}),
		}, newParsed())

		_, err := runtime.newService(loaded)
		if err == nil || !errors.Is(err, errRuntimeOpenSecretAPI) {
			t.Fatalf("expected open secret api error, got %v", err)
		}
	})

	t.Run("InitServiceError", func(t *testing.T) {
		runtime := newCommandRuntime(commandContext{
			stdout: &bytes.Buffer{},
			stderr: &bytes.Buffer{},
			deps: baseDeps(func(config.Config, string) (SecretAPI, error) {
				return nil, nil
			}),
		}, newParsed())

		_, err := runtime.newService(loaded)
		if err == nil || !errors.Is(err, errRuntimeInitSecretSyncError) {
			t.Fatalf("expected init service error, got %v", err)
		}
	})
}
