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

type commandConfigPolicy int

const (
	commandConfigNone commandConfigPolicy = iota
	commandConfigValidated
	commandConfigProjectOnly
)

type commandDef struct {
	Name      string
	Summary   string
	Flags     []commandFlagDef
	Doc       commandDoc
	Config    commandConfigPolicy
	RunParsed func(commandContext, *parsedCommand) int
}

var registeredCommandDefs = []commandDef{
	versionCommandDef,
	listCommandDef,
	pullCommandDef,
	pushCommandDef,
}

func commandDefs() []commandDef {
	out := make([]commandDef, len(registeredCommandDefs))
	copy(out, registeredCommandDefs)
	return out
}

func commandForName(name string) (commandDef, bool) {
	for _, def := range registeredCommandDefs {
		if def.Name == name {
			return def, true
		}
	}
	return commandDef{}, false
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
