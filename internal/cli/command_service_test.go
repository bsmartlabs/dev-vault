package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func TestCommandServiceModule_NewServiceWiresDeps(t *testing.T) {
	loaded := &config.Loaded{}
	api := newFakeSecretAPI()
	deps := Dependencies{
		Now:      func() time.Time { return time.Unix(123, 0) },
		Hostname: func() (string, error) { return "host", nil },
	}
	svc := newCommandService(loaded, api, deps)
	if svc.loaded != loaded || svc.api != api {
		t.Fatalf("service wiring mismatch: %#v", svc)
	}
	if got := svc.now(); got.Unix() != 123 {
		t.Fatalf("unexpected now value: %v", got)
	}
	h, err := svc.hostname()
	if err != nil || h != "host" {
		t.Fatalf("unexpected hostname: %q err=%v", h, err)
	}
}

func TestCommandService_ResolvePushSecret_InvalidMappingType(t *testing.T) {
	loaded := &config.Loaded{
		Cfg: config.Config{
			Region:    "fr-par",
			ProjectID: "project",
		},
	}
	svc := newCommandService(loaded, newFakeSecretAPI(), Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.resolvePushSecret("x-dev", config.MappingEntry{
		Path: "/",
		Type: "not-a-valid-type",
	}, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid mapping.type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
