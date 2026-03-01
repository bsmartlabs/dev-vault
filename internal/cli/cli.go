package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
)

type Dependencies struct {
	Version string
	Commit  string
	Date    string

	OpenSecretAPI func(cfg config.Config, profileOverride string) (SecretAPI, error)

	Now      func() time.Time
	Hostname func() (string, error)
	Getwd    func() (string, error)
}

func DefaultDependencies(version, commit, date string) Dependencies {
	return Dependencies{
		Version:       version,
		Commit:        commit,
		Date:          date,
		OpenSecretAPI: OpenScalewaySecretAPI,
		Now:           time.Now,
		Hostname:      os.Hostname,
		Getwd:         os.Getwd,
	}
}

func Run(args []string, stdout, stderr io.Writer, deps Dependencies) int {
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil || deps.Getwd == nil {
		fmt.Fprintln(stderr, "internal error: missing dependencies")
		return 1
	}
	if len(args) == 0 {
		printMainUsage(stderr)
		return 2
	}
	if len(args) > 1 && (args[1] == "-h" || args[1] == "--help") {
		printMainUsage(stdout)
		return 0
	}

	global := flag.NewFlagSet("dev-vault", flag.ContinueOnError)
	global.SetOutput(stderr)
	configPath := ""
	profileOverride := ""
	bindGlobalOptionFlags(global, &configPath, &profileOverride)

	global.Usage = func() {
		printMainUsage(stderr)
	}

	if err := global.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rest := global.Args()
	if len(rest) == 0 {
		printMainUsage(stderr)
		return 2
	}

	cmd := rest[0]
	ctx := commandContext{
		stdout:          stdout,
		stderr:          stderr,
		configPath:      configPath,
		profileOverride: profileOverride,
		deps:            deps,
	}
	switch cmd {
	case "help":
		if len(rest) > 1 {
			switch rest[1] {
			case "list":
				printListUsage(stdout)
				return 0
			case "pull":
				printPullUsage(stdout)
				return 0
			case "push":
				printPushUsage(stdout)
				return 0
			case "version":
				printVersionUsage(stdout)
				return 0
			default:
				fmt.Fprintf(stderr, "unknown command for help: %s\n", rest[1])
				printMainUsage(stderr)
				return 2
			}
		}
		printMainUsage(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "dev-vault %s (commit=%s date=%s)\n", deps.Version, deps.Commit, deps.Date)
		return 0
	case "list":
		return runList(ctx, rest[1:])
	case "pull":
		return runPull(ctx, rest[1:])
	case "push":
		return runPush(ctx, rest[1:])
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		printMainUsage(stderr)
		return 2
	}
}
