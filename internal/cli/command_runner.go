package cli

import (
	"flag"
)

type parsedCommand struct {
	fs              *flag.FlagSet
	configPath      string
	profileOverride string
	configPolicy    commandConfigPolicy
	pullOptions     pullOptions
	pushOptions     pushOptions
	listOptions     listOptions
}

type parsedFlagValues struct {
	boolValues   map[string]*bool
	stringValues map[string]*string
	sliceValues  map[string]*stringSliceFlag
}

func parseCommand(ctx commandContext, argv []string, def commandDef) (*parsedCommand, error) {
	if hasHelpFlag(argv) {
		if err := printCommandUsage(ctx.stdout, def); err != nil {
			return nil, outputError(err)
		}
		return nil, helpError(flag.ErrHelp)
	}

	fs := flag.NewFlagSet(def.Name, flag.ContinueOnError)
	fs.SetOutput(ctx.stderr)
	fs.Usage = func() {
		_ = printCommandUsage(ctx.stderr, def)
	}

	configPath := ctx.configPath
	profileOverride := ctx.profileOverride

	bindGlobalOptionFlags(fs, &configPath, &profileOverride)

	boolHolders := make(map[string]*bool, len(def.Flags))
	stringHolders := make(map[string]*string, len(def.Flags))
	sliceHolders := make(map[string]*stringSliceFlag, len(def.Flags))

	for _, flagDef := range def.Flags {
		switch flagDef.Kind {
		case commandFlagBool:
			value := false
			boolHolders[flagDef.Name] = &value
			fs.BoolVar(boolHolders[flagDef.Name], flagDef.Name, false, flagDef.Help)
		case commandFlagString:
			value := ""
			stringHolders[flagDef.Name] = &value
			fs.StringVar(stringHolders[flagDef.Name], flagDef.Name, "", flagDef.Help)
		case commandFlagStringSlice:
			value := stringSliceFlag{}
			sliceHolders[flagDef.Name] = &value
			fs.Var(sliceHolders[flagDef.Name], flagDef.Name, flagDef.Help)
		}
	}

	reordered := reorderFlags(argv, withGlobalFlagSpecs(takesValueMap(def)))
	if err := fs.Parse(reordered); err != nil {
		return nil, usageError(err)
	}

	parsed := &parsedCommand{
		fs:              fs,
		configPath:      configPath,
		profileOverride: profileOverride,
		configPolicy:    def.Config,
	}
	if def.DecodeParsed != nil {
		def.DecodeParsed(parsed, parsedFlagValues{
			boolValues:   boolHolders,
			stringValues: stringHolders,
			sliceValues:  sliceHolders,
		})
	}
	return parsed, nil
}

func hasHelpFlag(argv []string) bool {
	for _, arg := range argv {
		if arg == "--" {
			return false
		}
		if arg == "-h" || arg == "--help" || arg == "-help" {
			return true
		}
	}
	return false
}

func runParsedCommand(ctx commandContext, argv []string, def commandDef, run func(parsed *parsedCommand) int) int {
	parsed, parseErr := parseCommand(ctx, argv, def)
	if parseErr != nil {
		return exitCodeForError(parseErr)
	}
	return run(parsed)
}

func runCommand(ctx commandContext, argv []string, def commandDef) int {
	return runParsedCommand(ctx, argv, def, func(parsed *parsedCommand) int {
		return def.RunParsed(ctx, parsed)
	})
}

func boolFlagValue(values parsedFlagValues, name string) bool {
	value, ok := values.boolValues[name]
	return ok && value != nil && *value
}

func stringFlagValue(values parsedFlagValues, name string) string {
	value, ok := values.stringValues[name]
	if !ok || value == nil {
		return ""
	}
	return *value
}

func sliceFlagValue(values parsedFlagValues, name string) []string {
	value, ok := values.sliceValues[name]
	if !ok || value == nil || len(*value) == 0 {
		return nil
	}
	items := make([]string, len(*value))
	copy(items, *value)
	return items
}
