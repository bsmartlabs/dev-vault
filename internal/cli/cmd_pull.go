package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

func runPull(argv []string, stdout, stderr io.Writer, configPath, profileOverride string, deps Dependencies) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printPullUsage(stderr) }

	cfgPath := configPath
	prof := profileOverride

	var all bool
	var overwrite bool
	bindGlobalOptionFlags(fs, &cfgPath, &prof)
	fs.BoolVar(&all, "all", false, "Pull all mapping entries with mode pull|both (mode defaults to both)")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	argv = reorderFlags(argv, withGlobalFlagSpecs(map[string]bool{
		"all":       false,
		"overwrite": false,
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

	targets, err := selectMappingTargets(runtime.loaded.Cfg.Mapping, all, fs.Args(), "pull")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}

	results, err := runtime.service.pull(targets, overwrite)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	for _, item := range results {
		fmt.Fprintf(stdout, "pulled %s -> %s (rev=%d type=%s)\n", item.Name, item.File, item.Revision, item.Type)
	}
	return 0
}
