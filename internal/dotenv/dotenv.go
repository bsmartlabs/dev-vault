package dotenv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func Parse(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '='", lineNum)
		}
		key := strings.TrimSpace(line[:eq])
		if !isValidKey(key) {
			return nil, fmt.Errorf("line %d: invalid key %q", lineNum, key)
		}

		rawVal := strings.TrimSpace(line[eq+1:])
		if rawVal == "" {
			out[key] = ""
			continue
		}

		val, err := parseValue(rawVal)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func Render(env map[string]string) []byte {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteByte('"')
		b.WriteString(escapeDoubleQuoted(env[k]))
		b.WriteByte('"')
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func isValidKey(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case i == 0 && (unicode.IsLetter(r) || r == '_'):
			// ok
		case i > 0 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'):
			// ok
		default:
			return false
		}
	}
	return true
}

func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw[0] == '"' {
		return parseDoubleQuoted(raw)
	}
	if raw[0] == '\'' {
		return parseSingleQuoted(raw)
	}
	return strings.TrimSpace(raw), nil
}

func parseSingleQuoted(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '\'' {
		return "", errors.New("not single quoted")
	}
	// Find ending quote.
	for i := 1; i < len(raw); i++ {
		if raw[i] == '\'' {
			return raw[1:i], nil
		}
	}
	return "", errors.New("unterminated single-quoted value")
}

func parseDoubleQuoted(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' {
		return "", errors.New("not double quoted")
	}
	var b strings.Builder
	escaped := false
	for i := 1; i < len(raw); i++ {
		ch := raw[i]
		if escaped {
			switch ch {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				// Keep unknown escapes literal.
				b.WriteByte('\\')
				b.WriteByte(ch)
			}
			escaped = false
			continue
		}
		switch ch {
		case '\\':
			escaped = true
		case '"':
			return b.String(), nil
		default:
			b.WriteByte(ch)
		}
	}
	return "", errors.New("unterminated double-quoted value")
}

func escapeDoubleQuoted(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
