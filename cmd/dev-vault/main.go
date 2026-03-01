package main

import (
	"io"
	"os"

	"github.com/bsmartlabs/dev-vault/internal/cli"
	scwprovider "github.com/bsmartlabs/dev-vault/internal/secretprovider/scaleway"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	exitFn  = os.Exit
)

func main() {
	exitFn(runMain(os.Args, os.Stdout, os.Stderr, version, commit, date, cli.Run))
}

func runMain(args []string, stdout, stderr io.Writer, version, commit, date string, runFn func([]string, io.Writer, io.Writer, cli.Dependencies) int) int {
	deps := cli.DefaultDependencies(version, commit, date, scwprovider.Open)
	return runFn(args, stdout, stderr, deps)
}
