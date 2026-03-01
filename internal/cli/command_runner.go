package cli

import (
	"errors"
	"flag"
)

type parsedCommand struct {
	fs              *flag.FlagSet
	configPath      string
	profileOverride string
	boolValues      map[string]bool
	stringValues    map[string]string
	sliceValues     map[string][]string
}

func (p *parsedCommand) Bool(name string) bool {
	return p.boolValues[name]
}

func (p *parsedCommand) String(name string) string {
	return p.stringValues[name]
}

func (p *parsedCommand) Strings(name string) []string {
	values := p.sliceValues[name]
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

type parseCommandError struct {
	code int
	err  error
}

func (e *parseCommandError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return "command parse exit"
}

func (e *parseCommandError) Unwrap() error {
	return e.err
}

func parseCommandExitCode(err error) (code int, terminal bool) {
	if err == nil {
		return 0, false
	}
	var parseErr *parseCommandError
	if errors.As(err, &parseErr) {
		return parseErr.code, true
	}
	return 1, true
}

func parseCommand(ctx commandContext, argv []string, def commandDef) (*parsedCommand, error) {
	fs := flag.NewFlagSet(def.Name, flag.ContinueOnError)
	fs.SetOutput(ctx.stderr)
	fs.Usage = func() { printCommandUsage(ctx.stderr, def) }

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
		if errors.Is(err, flag.ErrHelp) {
			return nil, &parseCommandError{code: 0, err: err}
		}
		return nil, &parseCommandError{code: 2, err: err}
	}

	boolValues := make(map[string]bool, len(boolHolders))
	for name, value := range boolHolders {
		boolValues[name] = *value
	}
	stringValues := make(map[string]string, len(stringHolders))
	for name, value := range stringHolders {
		stringValues[name] = *value
	}
	sliceValues := make(map[string][]string, len(sliceHolders))
	for name, value := range sliceHolders {
		if len(*value) == 0 {
			continue
		}
		items := make([]string, len(*value))
		copy(items, *value)
		sliceValues[name] = items
	}

	return &parsedCommand{
		fs:              fs,
		configPath:      configPath,
		profileOverride: profileOverride,
		boolValues:      boolValues,
		stringValues:    stringValues,
		sliceValues:     sliceValues,
	}, nil
}

func runParsedCommand(ctx commandContext, argv []string, def commandDef, run func(parsed *parsedCommand) int) int {
	parsed, parseErr := parseCommand(ctx, argv, def)
	if code, terminal := parseCommandExitCode(parseErr); terminal {
		return code
	}
	return run(parsed)
}

func runCommand(ctx commandContext, argv []string, def commandDef) int {
	return runParsedCommand(ctx, argv, def, func(parsed *parsedCommand) int {
		return def.RunParsed(ctx, parsed)
	})
}
