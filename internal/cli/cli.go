package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type SecretAPI interface {
	ListSecrets(req *secret.ListSecretsRequest, opts ...scw.RequestOption) (*secret.ListSecretsResponse, error)
	AccessSecretVersion(req *secret.AccessSecretVersionRequest, opts ...scw.RequestOption) (*secret.AccessSecretVersionResponse, error)
	CreateSecret(req *secret.CreateSecretRequest, opts ...scw.RequestOption) (*secret.Secret, error)
	CreateSecretVersion(req *secret.CreateSecretVersionRequest, opts ...scw.RequestOption) (*secret.SecretVersion, error)
}

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
		return runList(rest[1:], stdout, stderr, configPath, profileOverride, deps)
	case "pull":
		return runPull(rest[1:], stdout, stderr, configPath, profileOverride, deps)
	case "push":
		return runPush(rest[1:], stdout, stderr, configPath, profileOverride, deps)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		printMainUsage(stderr)
		return 2
	}
}
