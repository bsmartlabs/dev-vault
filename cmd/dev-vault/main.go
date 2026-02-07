package main

import (
	"os"

	"github.com/bsmartlabs/dev-vault/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	osExit = os.Exit
	run    = cli.Run
)

func main() {
	deps := cli.DefaultDependencies(version, commit, date)
	osExit(run(os.Args, os.Stdout, os.Stderr, deps))
}
