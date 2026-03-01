package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

type usageWriter struct {
	w   io.Writer
	err error
}

func (u *usageWriter) line(a ...any) {
	if u.err != nil {
		return
	}
	_, u.err = fmt.Fprintln(u.w, a...)
}

func (u *usageWriter) f(format string, a ...any) {
	if u.err != nil {
		return
	}
	_, u.err = fmt.Fprintf(u.w, format, a...)
}

func printMainUsage(w io.Writer) error {
	out := usageWriter{w: w}
	out.line("dev-vault")
	out.line("  Pull/push Scaleway Secret Manager secrets to disk for local development.")
	out.line()
	out.line("Usage:")
	out.line("  dev-vault [global options] <command> [command options] [args...]")
	out.line("  dev-vault help [command]")
	out.line()
	out.line("Global options:")
	out.f("  --config <path>   Path to %s. If omitted: search upward from cwd.\n", config.DefaultConfigName)
	out.line("  --profile <name>  Scaleway profile override (uses ~/.config/scw/config.yaml)")
	out.line()
	out.line("Commands:")
	for _, def := range commandDefs {
		out.f("  %-8s %s\n", def.Name, def.Summary)
	}
	out.line()
	out.line("Hard safety constraints:")
	out.line("  - Refuses to operate on secret names that do not end with '-dev'.")
	out.line("  - Never prints secret payloads.")
	out.line("  - Pull writes files atomically and chmods them to 0600 (on Unix).")
	out.line()
	out.line("Batch behavior:")
	out.line("  - mapping.mode defaults to both.")
	out.line("  - pull --all includes mapping entries with mapping.mode in {pull, both}.")
	out.line("  - push --all includes mapping entries with mapping.mode in {push, both}.")
	out.f("  - %s\n", explicitModePolicySentence)
	out.line("  - Note: mapping.mode='sync' is accepted as a legacy alias for 'both'.")
	out.line()
	out.line("Examples:")
	out.line("  dev-vault list --json")
	out.line("  dev-vault pull bweb-env-bsmart-dev --overwrite")
	out.line("  dev-vault push bweb-env-bsmart-dev")
	out.line("  dev-vault pull --config .scw.json bweb-env-bsmart-dev --overwrite")
	out.line()
	out.line("Notes for automation/LLMs:")
	out.line("  - Global options can be passed either before the command or as command options (e.g. 'pull --config ...').")
	out.line("  - Exit codes: 0=success, 1=runtime error, 2=usage error.")
	return out.err
}

func printCommandUsage(w io.Writer, def commandDef) error {
	out := usageWriter{w: w}
	out.line("Usage:")
	out.f("  %s\n", def.Doc.Synopsis)

	if len(def.Doc.Description) > 0 {
		out.line()
		for _, line := range def.Doc.Description {
			out.line(line)
		}
	}

	if len(def.Flags) > 0 {
		out.line()
		out.line("Options:")
		for _, flagDef := range sortedFlagDefs(def.Flags) {
			out.f("  --%s\n", formatFlagUsage(flagDef))
		}
	}

	if len(def.Doc.Notes) > 0 {
		out.line()
		out.line("Notes:")
		for _, note := range def.Doc.Notes {
			out.line("  - " + note)
		}
	}

	if len(def.Doc.Examples) > 0 {
		out.line()
		out.line("Examples:")
		for _, example := range def.Doc.Examples {
			out.f("  %s\n", example)
		}
	}

	return out.err
}

func sortedFlagDefs(flags []commandFlagDef) []commandFlagDef {
	out := make([]commandFlagDef, len(flags))
	copy(out, flags)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func formatFlagUsage(flagDef commandFlagDef) string {
	out := flagDef.Name
	if flagDef.Kind != commandFlagBool {
		out += " " + flagDef.ValueName
	}
	if flagDef.Help != "" {
		out += "  " + flagDef.Help
	}
	return out
}

func printVersionUsage(w io.Writer) error {
	return printCommandUsage(w, versionCommandDef)
}

func printListUsage(w io.Writer) error {
	return printCommandUsage(w, listCommandDef)
}

func printPullUsage(w io.Writer) error {
	return printCommandUsage(w, pullCommandDef)
}

func printPushUsage(w io.Writer) error {
	return printCommandUsage(w, pushCommandDef)
}
