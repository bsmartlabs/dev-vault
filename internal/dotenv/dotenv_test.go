package dotenv

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	got, err := Parse([]byte(strings.Join([]string{
		"# comment",
		"export FOO=\"bar\"",
		"BAZ=qux",
		"EMPTY=",
		"SINGLE='a b'",
		`ESC="a\n\t\\\"b"`,
		"",
	}, "\n")))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	want := map[string]string{
		"FOO":    "bar",
		"BAZ":    "qux",
		"EMPTY":  "",
		"SINGLE": "a b",
		"ESC":    "a\n\t\\\"b",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected map\nwant=%#v\ngot =%#v", want, got)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		name string
		in   string
		sub  string
	}{
		{"MissingEquals", "NOPE", "missing"},
		{"InvalidKey", "1BAD=x", "invalid key"},
		{"UnterminatedSingle", "A='x", "unterminated"},
		{"UnterminatedDouble", `A="x`, "unterminated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.in))
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("expected error containing %q, got %v", tc.sub, err)
			}
		})
	}
}

func TestRender(t *testing.T) {
	out := Render(map[string]string{
		"B": "b",
		"A": "a\n\"x\"\\",
	})
	if !bytes.HasPrefix(out, []byte("A=")) {
		t.Fatalf("expected sorted keys, got:\n%s", string(out))
	}
	if !bytes.Contains(out, []byte(`A="a\n\"x\"\\"
`)) {
		t.Fatalf("unexpected escaping:\n%s", string(out))
	}
}

func TestRoundTrip(t *testing.T) {
	env := map[string]string{
		"A": "x",
		"B": "y\nz",
	}
	rendered := Render(env)
	parsed, err := Parse(rendered)
	if err != nil {
		t.Fatalf("parse rendered: %v", err)
	}
	if !reflect.DeepEqual(parsed, env) {
		t.Fatalf("roundtrip mismatch\nwant=%#v\ngot =%#v", env, parsed)
	}
}

func TestHelpersAndScannerError(t *testing.T) {
	// isValidKey branches.
	if isValidKey("") {
		t.Fatalf("expected empty key to be invalid")
	}
	if !isValidKey("A1") {
		t.Fatalf("expected key with digit after first char to be valid")
	}
	if !isValidKey("_A") {
		t.Fatalf("expected underscore-start key to be valid")
	}
	if isValidKey("A-B") {
		t.Fatalf("expected dash in key to be invalid")
	}

	// parseValue branches.
	if v, err := parseValue(""); err != nil || v != "" {
		t.Fatalf("parseValue empty: v=%q err=%v", v, err)
	}
	if v, err := parseValue("  x  "); err != nil || v != "x" {
		t.Fatalf("parseValue trim: v=%q err=%v", v, err)
	}

	// parseSingleQuoted error + empty string.
	if _, err := parseSingleQuoted("x"); err == nil {
		t.Fatalf("expected error for non-single-quoted input")
	}
	if v, err := parseSingleQuoted("''"); err != nil || v != "" {
		t.Fatalf("parseSingleQuoted empty: v=%q err=%v", v, err)
	}

	// parseDoubleQuoted error + unknown escape + \\r escape.
	if _, err := parseDoubleQuoted("x"); err == nil {
		t.Fatalf("expected error for non-double-quoted input")
	}
	if v, err := parseDoubleQuoted("\"\\q\""); err != nil || v != "\\q" {
		t.Fatalf("parseDoubleQuoted unknown escape: v=%q err=%v", v, err)
	}
	if v, err := parseDoubleQuoted("\"a\\rb\""); err != nil || v != "a\rb" {
		t.Fatalf("parseDoubleQuoted \\r: v=%q err=%v", v, err)
	}

	// escapeDoubleQuoted \\r and \\t.
	escaped := escapeDoubleQuoted("a\rb\tc")
	if !strings.Contains(escaped, `\r`) || !strings.Contains(escaped, `\t`) {
		t.Fatalf("expected \\r and \\t escapes, got %q", escaped)
	}

	// Scanner error branch: line exceeds bufio.Scanner default token size.
	long := strings.Repeat("a", 70*1024)
	_, err := Parse([]byte("KEY=" + long))
	if err == nil {
		t.Fatalf("expected scanner error")
	}
}
