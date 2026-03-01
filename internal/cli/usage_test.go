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
		fn       func(io.Writer)
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
			tc.fn(&buf)
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
	printMainUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "must satisfy mapping.mode") {
		t.Fatalf("expected main usage to mention explicit names must satisfy mapping.mode, got %q", out)
	}
}
