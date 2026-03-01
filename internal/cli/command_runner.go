package cli

import (
	"errors"
	"flag"
	"io"
)

type commandSpec struct {
	name           string
	usage          func(io.Writer)
	localFlagSpecs map[string]bool
	bindFlags      func(fs *flag.FlagSet, configPath *string, profileOverride *string)
}

type parsedCommand struct {
	fs              *flag.FlagSet
	configPath      string
	profileOverride string
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

func parseCommand(ctx commandContext, argv []string, spec commandSpec) (*parsedCommand, error) {
	fs := flag.NewFlagSet(spec.name, flag.ContinueOnError)
	fs.SetOutput(ctx.stderr)
	fs.Usage = func() { spec.usage(ctx.stderr) }

	configPath := ctx.configPath
	profileOverride := ctx.profileOverride

	bindGlobalOptionFlags(fs, &configPath, &profileOverride)
	if spec.bindFlags != nil {
		spec.bindFlags(fs, &configPath, &profileOverride)
	}

	reordered := reorderFlags(argv, withGlobalFlagSpecs(spec.localFlagSpecs))
	if err := fs.Parse(reordered); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, &parseCommandError{code: 0, err: err}
		}
		return nil, &parseCommandError{code: 2, err: err}
	}

	return &parsedCommand{
		fs:              fs,
		configPath:      configPath,
		profileOverride: profileOverride,
	}, nil
}
