package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type Dependencies struct {
	Version string
	Commit  string
	Date    string

	OpenSecretAPI func(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error)

	Now      func() time.Time
	Hostname func() (string, error)
	Getwd    func() (string, error)
}

func DefaultDependencies(version, commit, date string, openSecretAPI func(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error)) Dependencies {
	return Dependencies{
		Version:       version,
		Commit:        commit,
		Date:          date,
		OpenSecretAPI: openSecretAPI,
		Now:           time.Now,
		Hostname:      os.Hostname,
		Getwd:         os.Getwd,
	}
}

func Run(args []string, stdout, stderr io.Writer, deps Dependencies) int {
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil || deps.Getwd == nil {
		if _, err := fmt.Fprintln(stderr, "internal error: missing dependencies"); err != nil {
			return 1
		}
		return 1
	}
	if len(args) == 0 {
		if err := printMainUsage(stderr); err != nil {
			return 1
		}
		return 2
	}
	if len(args) > 1 && (args[1] == "-h" || args[1] == "--help") {
		if err := printMainUsage(stdout); err != nil {
			return 1
		}
		return 0
	}

	global := flag.NewFlagSet("dev-vault", flag.ContinueOnError)
	global.SetOutput(stderr)
	configPath := ""
	profileOverride := ""
	bindGlobalOptionFlags(global, &configPath, &profileOverride)

	global.Usage = func() {
		_ = printMainUsage(stderr)
	}

	if err := global.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rest := global.Args()
	if len(rest) == 0 {
		if err := printMainUsage(stderr); err != nil {
			return 1
		}
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
			usagePrinter, ok := usageForCommand(rest[1])
			if !ok {
				if _, err := fmt.Fprintf(stderr, "unknown command for help: %s\n", rest[1]); err != nil {
					return 1
				}
				if err := printMainUsage(stderr); err != nil {
					return 1
				}
				return 2
			}
			if err := usagePrinter(stdout); err != nil {
				return 1
			}
			return 0
		}
		if err := printMainUsage(stdout); err != nil {
			return 1
		}
		return 0
	default:
		def, ok := commandForName(cmd)
		if !ok {
			if _, err := fmt.Fprintf(stderr, "unknown command: %s\n", cmd); err != nil {
				return 1
			}
			if err := printMainUsage(stderr); err != nil {
				return 1
			}
			return 2
		}
		return runCommand(ctx, rest[1:], def)
	}
}
