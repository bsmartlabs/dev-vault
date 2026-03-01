package cli

import (
	"bytes"
	"testing"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestRunPush_HelpSmoke(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := runPush([]string{"-h"}, &out, &errBuf, "", "", baseDeps(func(cfg config.Config, s string) (SecretAPI, error) {
		return nil, nil
	}))
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}
