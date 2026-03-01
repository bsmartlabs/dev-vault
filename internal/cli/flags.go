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
			positional = append(positional, argv[i+1:]...)
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

func withGlobalFlagSpecs(spec map[string]bool) map[string]bool {
	out := make(map[string]bool, len(spec)+2)
	out["config"] = true
	out["profile"] = true
	for key, value := range spec {
		out[key] = value
	}
	return out
}
