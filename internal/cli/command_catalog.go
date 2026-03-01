package cli

import "io"

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

var commandDefs = []commandDef{
	versionCommandDef,
	listCommandDef,
	pullCommandDef,
	pushCommandDef,
}

func commandForName(name string) (commandDef, bool) {
	for _, def := range commandDefs {
		if def.Name == name {
			return def, true
		}
	}
	return commandDef{}, false
}

func usageForCommand(name string) (func(io.Writer), bool) {
	def, ok := commandForName(name)
	if !ok {
		return nil, false
	}
	return func(w io.Writer) {
		printCommandUsage(w, def)
	}, true
}

func takesValueMap(def commandDef) map[string]bool {
	spec := make(map[string]bool, len(def.Flags))
	for _, flagDef := range def.Flags {
		spec[flagDef.Name] = flagDef.Kind != commandFlagBool
	}
	return spec
}
