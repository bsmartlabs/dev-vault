package cli

import (
	"fmt"
	"io"
)

type commandFlagKind int

const (
	commandFlagBool commandFlagKind = iota + 1
	commandFlagString
	commandFlagStringSlice
)

type commandFlagDef struct {
	Name      string
	Kind      commandFlagKind
	ValueName string
	Help      string
}

type commandDoc struct {
	Synopsis    string
	Description []string
	Notes       []string
	Examples    []string
}

type commandDef struct {
	Name      string
	Summary   string
	Flags     []commandFlagDef
	Doc       commandDoc
	RunParsed func(commandContext, *parsedCommand) int
}

type commandCatalogState struct {
	ordered []commandDef
	byName  map[string]commandDef
}

var commandCatalog = newCommandCatalog(
	versionCommandDef,
	listCommandDef,
	pullCommandDef,
	pushCommandDef,
)

func newCommandCatalog(defs ...commandDef) commandCatalogState {
	state := commandCatalogState{
		ordered: make([]commandDef, 0, len(defs)),
		byName:  make(map[string]commandDef, len(defs)),
	}
	for _, def := range defs {
		if def.Name == "" {
			panic("cli command catalog: empty command name")
		}
		if def.RunParsed == nil {
			panic(fmt.Sprintf("cli command catalog: command %q missing RunParsed", def.Name))
		}
		if _, exists := state.byName[def.Name]; exists {
			panic(fmt.Sprintf("cli command catalog: duplicate command name %q", def.Name))
		}
		state.byName[def.Name] = def
		state.ordered = append(state.ordered, def)
	}
	return state
}

func commandDefs() []commandDef {
	out := make([]commandDef, len(commandCatalog.ordered))
	copy(out, commandCatalog.ordered)
	return out
}

func commandForName(name string) (commandDef, bool) {
	def, ok := commandCatalog.byName[name]
	return def, ok
}

func usageForCommand(name string) (func(io.Writer) error, bool) {
	def, ok := commandForName(name)
	if !ok {
		return nil, false
	}
	return func(w io.Writer) error {
		return printCommandUsage(w, def)
	}, true
}

func takesValueMap(def commandDef) map[string]bool {
	spec := make(map[string]bool, len(def.Flags))
	for _, flagDef := range def.Flags {
		spec[flagDef.Name] = flagDef.Kind != commandFlagBool
	}
	return spec
}
