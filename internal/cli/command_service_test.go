package cli

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/mapping"
)

func TestCommandServiceModule_NewServiceWiresDeps(t *testing.T) {
	loaded := &config.Loaded{}
	api := newFakeSecretAPI()
	deps := Dependencies{
		Now:      func() time.Time { return time.Unix(123, 0) },
		Hostname: func() (string, error) { return "host", nil },
	}
	svc := newCommandService(loaded, api, deps)
	if svc.api == nil {
		t.Fatalf("service api should be initialized: %#v", svc)
	}
	if got := svc.now(); got.Unix() != 123 {
		t.Fatalf("unexpected now value: %v", got)
	}
	h, err := svc.hostname()
	if err != nil || h != "host" {
		t.Fatalf("unexpected hostname: %q err=%v", h, err)
	}
}

func TestCommandService_LookupMappedSecret_ListError(t *testing.T) {
	api := newFakeSecretAPI()
	api.ListErr = errors.New("boom")
	svc := newCommandServiceWithConfig(commandServiceConfig{}, api, Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.lookupMappedSecret("x-dev", mapping.Entry{Path: "/"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list secrets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandService_ResolveMappedSecret_NoCreateMissing(t *testing.T) {
	svc := newCommandServiceWithConfig(commandServiceConfig{}, newFakeSecretAPI(), Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.resolveMappedSecret("x-dev", mapping.Entry{Path: "/"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "secret not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandService_List_ListError(t *testing.T) {
	api := newFakeSecretAPI()
	api.ListErr = errors.New("boom")
	svc := newCommandServiceWithConfig(commandServiceConfig{}, api, Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.list(listQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandService_ResolveMappedSecret_CreateMissingRequiresType(t *testing.T) {
	svc := newCommandServiceWithConfig(commandServiceConfig{}, newFakeSecretAPI(), Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.resolveMappedSecret("x-dev", mapping.Entry{Path: "/"}, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "create-missing requires mapping.type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandService_ResolveMappedSecret_ListErrorWithCreateMissing(t *testing.T) {
	api := newFakeSecretAPI()
	api.ListErr = errors.New("boom")
	svc := newCommandServiceWithConfig(commandServiceConfig{}, api, Dependencies{
		Now:      time.Now,
		Hostname: func() (string, error) { return "host", nil },
	})

	_, err := svc.resolveMappedSecret("x-dev", mapping.Entry{Path: "/", Type: "opaque"}, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "list secrets") {
		t.Fatalf("unexpected error: %v", err)
	}
}
