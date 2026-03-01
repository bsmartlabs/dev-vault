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

func parseCommand(ctx commandContext, argv []string, spec commandSpec) (*parsedCommand, int, error) {
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
			return nil, 0, nil
		}
		return nil, 2, usageError(err)
	}

	return &parsedCommand{
		fs:              fs,
		configPath:      configPath,
		profileOverride: profileOverride,
	}, -1, nil
}
