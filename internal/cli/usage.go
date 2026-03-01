package cli

import (
	"fmt"
	"io"
	"sort"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

func writeLine(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

func writef(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func printMainUsage(w io.Writer) {
	writeLine(w, "dev-vault")
	writeLine(w, "  Pull/push Scaleway Secret Manager secrets to disk for local development.")
	writeLine(w)
	writeLine(w, "Usage:")
	writeLine(w, "  dev-vault [global options] <command> [command options] [args...]")
	writeLine(w, "  dev-vault help [command]")
	writeLine(w)
	writeLine(w, "Global options:")
	writef(w, "  --config <path>   Path to %s. If omitted: search upward from cwd.\n", config.DefaultConfigName)
	writeLine(w, "  --profile <name>  Scaleway profile override (uses ~/.config/scw/config.yaml)")
	writeLine(w)
	writeLine(w, "Commands:")
	for _, def := range commandDefs {
		writef(w, "  %-8s %s\n", def.Name, def.Summary)
	}
	writeLine(w)
	writeLine(w, "Hard safety constraints:")
	writeLine(w, "  - Refuses to operate on secret names that do not end with '-dev'.")
	writeLine(w, "  - Never prints secret payloads.")
	writeLine(w, "  - Pull writes files atomically and chmods them to 0600 (on Unix).")
	writeLine(w)
	writeLine(w, "Batch behavior:")
	writeLine(w, "  - mapping.mode defaults to both.")
	writeLine(w, "  - pull --all includes mapping entries with mapping.mode in {pull, both}.")
	writeLine(w, "  - push --all includes mapping entries with mapping.mode in {push, both}.")
	writef(w, "  - %s\n", explicitModePolicySentence)
	writeLine(w, "  - Note: mapping.mode='sync' is accepted as a legacy alias for 'both'.")
	writeLine(w)
	writeLine(w, "Examples:")
	writeLine(w, "  dev-vault list --json")
	writeLine(w, "  dev-vault pull bweb-env-bsmart-dev --overwrite")
	writeLine(w, "  dev-vault push bweb-env-bsmart-dev")
	writeLine(w, "  dev-vault pull --config .scw.json bweb-env-bsmart-dev --overwrite")
	writeLine(w)
	writeLine(w, "Notes for automation/LLMs:")
	writeLine(w, "  - Global options can be passed either before the command or as command options (e.g. 'pull --config ...').")
	writeLine(w, "  - Exit codes: 0=success, 1=runtime error, 2=usage error.")
}

func printCommandUsage(w io.Writer, def commandDef) {
	writeLine(w, "Usage:")
	writef(w, "  %s\n", def.Doc.Synopsis)

	if len(def.Doc.Description) > 0 {
		writeLine(w)
		for _, line := range def.Doc.Description {
			writeLine(w, line)
		}
	}

	if len(def.Flags) > 0 {
		writeLine(w)
		writeLine(w, "Options:")
		for _, flagDef := range sortedFlagDefs(def.Flags) {
			writef(w, "  --%s\n", formatFlagUsage(flagDef))
		}
	}

	if len(def.Doc.Notes) > 0 {
		writeLine(w)
		writeLine(w, "Notes:")
		for _, note := range def.Doc.Notes {
			writeLine(w, "  - "+note)
		}
	}

	if len(def.Doc.Examples) > 0 {
		writeLine(w)
		writeLine(w, "Examples:")
		for _, example := range def.Doc.Examples {
			writef(w, "  %s\n", example)
		}
	}
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

func printVersionUsage(w io.Writer) {
	printCommandUsage(w, versionCommandDef)
}

func printListUsage(w io.Writer) {
	printCommandUsage(w, listCommandDef)
}

func printPullUsage(w io.Writer) {
	printCommandUsage(w, pullCommandDef)
}

func printPushUsage(w io.Writer) {
	printCommandUsage(w, pushCommandDef)
}
