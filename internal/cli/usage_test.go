package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestUsageFunctions_BasicSmoke(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(io.Writer) error
		contains string
	}{
		{name: "main", fn: printMainUsage, contains: "dev-vault"},
		{name: "version", fn: printVersionUsage, contains: "version"},
		{name: "list", fn: printListUsage, contains: "list [options]"},
		{name: "pull", fn: printPullUsage, contains: "pull (--all | <secret-dev> ...)"},
		{name: "push", fn: printPushUsage, contains: "push (--all | <secret-dev> ...)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tc.fn(&buf); err != nil {
				t.Fatalf("usage writer returned error: %v", err)
			}
			out := buf.String()
			if out == "" {
				t.Fatal("expected usage output")
			}
			if !strings.Contains(out, tc.contains) {
				t.Fatalf("expected output to contain %q, got %q", tc.contains, out)
			}
		})
	}
}

func TestPrintMainUsage_ExplicitNamesMustRespectMode(t *testing.T) {
	var buf bytes.Buffer
	if err := printMainUsage(&buf); err != nil {
		t.Fatalf("printMainUsage: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "must satisfy mapping.mode") {
		t.Fatalf("expected main usage to mention explicit names must satisfy mapping.mode, got %q", out)
	}
}

func TestUsageWriter_ShortCircuitOnError(t *testing.T) {
	w := &usageWriter{w: &failingWriter{}}
	w.line("first")
	if w.err == nil {
		t.Fatal("expected write error after first line")
	}
	w.line("second")
	w.f("format %s", "x")
}
