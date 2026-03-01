package cli

import (
	"flag"
	"strings"
)

const (
	globalConfigFlagUsage      = "Path to .scw.json (default: search upward from cwd)"
	globalProfileFlagUsage     = "Scaleway config profile override"
	explicitModePolicySentence = "Explicit pull/push names must satisfy mapping.mode for that command."
)

var globalOptionTakesValue = map[string]bool{
	"config":  true,
	"profile": true,
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func reorderFlags(argv []string, takesValue map[string]bool) []string {
	// Go's standard flag package stops parsing when it sees the first non-flag argument.
	// For a better CLI UX, accept flags after positional args by reordering them.
	var flags []string
	var positional []string

	normalize := func(tok string) string {
		tok = strings.TrimLeft(tok, "-")
		if i := strings.IndexByte(tok, '='); i >= 0 {
			tok = tok[:i]
		}
		return tok
	}

	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if tok == "--" {
			rest := argv[i+1:]
			keepSentinel := len(positional) == 0 &&
				len(rest) > 0 &&
				strings.HasPrefix(rest[0], "-") &&
				rest[0] != "-"
			if keepSentinel {
				positional = append(positional, "--")
			}
			positional = append(positional, rest...)
			break
		}
		if strings.HasPrefix(tok, "-") && tok != "-" {
			flags = append(flags, tok)
			name := normalize(tok)
			if takesValue[name] && !strings.Contains(tok, "=") && i+1 < len(argv) {
				flags = append(flags, argv[i+1])
				i++
			}
			continue
		}
		positional = append(positional, tok)
	}

	return append(flags, positional...)
}

func bindGlobalOptionFlags(fs *flag.FlagSet, configPath *string, profileOverride *string) {
	fs.StringVar(configPath, "config", *configPath, globalConfigFlagUsage)
	fs.StringVar(profileOverride, "profile", *profileOverride, globalProfileFlagUsage)
}

func isGlobalOptionFlag(name string) (bool, bool) {
	takesValue, ok := globalOptionTakesValue[name]
	return takesValue, ok
}

func parseLongFlagToken(token string) (name string, hasValue bool, ok bool) {
	if !strings.HasPrefix(token, "--") {
		return "", false, false
	}
	trimmed := strings.TrimPrefix(token, "--")
	if trimmed == "" {
		return "", false, false
	}
	if idx := strings.IndexByte(trimmed, '='); idx >= 0 {
		return trimmed[:idx], true, true
	}
	return trimmed, false, true
}

func withGlobalFlagSpecs(spec map[string]bool) map[string]bool {
	out := make(map[string]bool, len(spec)+2)
	for name, takesValue := range globalOptionTakesValue {
		out[name] = takesValue
	}
	for key, value := range spec {
		out[key] = value
	}
	return out
}
