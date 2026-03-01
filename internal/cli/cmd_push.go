package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

func runPush(argv []string, stdout, stderr io.Writer, configPath, profileOverride string, deps Dependencies) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printPushUsage(stderr) }

	cfgPath := configPath
	prof := profileOverride

	var all bool
	var yes bool
	var disablePrevious bool
	var description string
	var createMissing bool

	bindGlobalOptionFlags(fs, &cfgPath, &prof)
	fs.BoolVar(&all, "all", false, "Push all mapping entries with mode push|both (mode defaults to both)")
	fs.BoolVar(&yes, "yes", false, "Confirm batch push (required when pushing more than one secret)")
	fs.BoolVar(&disablePrevious, "disable-previous", false, "Disable previous enabled version when creating a new version")
	fs.StringVar(&description, "description", "", "Description for the new version (optional)")
	fs.BoolVar(&createMissing, "create-missing", false, "Create missing secrets (requires mapping.type)")

	argv = reorderFlags(argv, withGlobalFlagSpecs(map[string]bool{
		"all":              false,
		"yes":              false,
		"disable-previous": false,
		"description":      true,
		"create-missing":   false,
	}))
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	runtime, err := buildCommandRuntime(cfgPath, prof, deps)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	targets, err := selectMappingTargets(runtime.loaded.Cfg.Mapping, all, fs.Args(), "push")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}

	if len(targets) > 1 && !yes {
		fmt.Fprintln(stderr, "refusing to push multiple secrets without --yes")
		return 2
	}

	results, err := runtime.service.push(targets, pushOptions{
		Description:     description,
		DisablePrevious: disablePrevious,
		CreateMissing:   createMissing,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	for _, item := range results {
		fmt.Fprintf(stdout, "pushed %s (rev=%d)\n", item.Name, item.Revision)
	}
	return 0
}
