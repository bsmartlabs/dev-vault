package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/pathpolicy"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
)

type Dependencies struct {
	Version string
	Commit  string
	Date    string

	OpenSecretAPI func(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error)

	Now                func() time.Time
	Hostname           func() (string, error)
	ResolveProjectPath func(rootDir string, rel string) (string, error)
}

func DefaultDependencies(version, commit, date string, openSecretAPI func(cfg config.Config, profileOverride string) (secretprovider.SecretAPI, error)) Dependencies {
	return Dependencies{
		Version:            version,
		Commit:             commit,
		Date:               date,
		OpenSecretAPI:      openSecretAPI,
		Now:                time.Now,
		Hostname:           os.Hostname,
		ResolveProjectPath: pathpolicy.ResolveProjectFile,
	}
}

// normalizeDashHelp converts the Go-flag-style "-help" to pflag-style "--help"
// so that Cobra recognises it. Without this, pflag interprets "-help" as the
// concatenation of short flags -h -e -l -p and errors on the unknown letters.
func normalizeDashHelp(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	for i, a := range out {
		if a == "-help" {
			out[i] = "--help"
		}
		if a == "--" {
			break
		}
	}
	return out
}

func Run(args []string, stdout, stderr io.Writer, deps Dependencies) int {
	if len(args) == 0 {
		args = []string{"dev-vault"}
	}

	var configPath, profileOverride string

	// helpWriteErr captures write errors from Cobra's built-in -h/--help
	// rendering. Cobra's Help() swallows write errors (always returns nil),
	// so we use a custom HelpFunc to detect them.
	var helpWriteErr error

	rootCmd := &cobra.Command{
		Use:   "dev-vault",
		Short: "Pull/push Scaleway Secret Manager secrets to disk for local development.",
		Long: `dev-vault
  Pull/push Scaleway Secret Manager secrets to disk for local development.

Hard safety constraints:
  - Refuses to operate on secret names that do not end with '-dev'.
  - Never prints secret payloads.
  - Pull writes files atomically and chmods them to 0600 (on Unix).

Batch behavior:
  - mapping.mode is required and must be pull, push, or skip.
  - pull --all includes only mapping entries with mapping.mode=pull.
  - push --all includes only mapping entries with mapping.mode=push.
  - mapping.mode=skip is excluded from both pull --all and push --all.
  - Explicit pull/push names must satisfy mapping.mode for that command.

Notes for automation/LLMs:
  - Global options can be passed either before the command or as command options (e.g. 'pull --config ...').
  - Exit codes: 0=success, 1=runtime error, 2=usage error.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetOut(stderr)
			_ = cmd.Usage()
			return usageError(fmt.Errorf("expected a subcommand"))
		},
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(normalizeDashHelp(args[1:]))
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// Override Cobra's built-in HelpFunc so write errors are not swallowed.
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		usage := cmd.UsageString()
		if _, err := fmt.Fprint(cmd.OutOrStdout(), usage); err != nil {
			helpWriteErr = err
		}
	})

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "",
		fmt.Sprintf("Path to %s (default: search upward from cwd)", config.DefaultConfigName))
	rootCmd.PersistentFlags().StringVar(&profileOverride, "profile", "",
		"Scaleway config profile override (uses ~/.config/scw/config.yaml)")

	rootCmd.AddCommand(
		newVersionCmd(deps, stdout),
		newListCmd(deps, stdout, stderr, &configPath, &profileOverride),
		newPullCmd(deps, stdout, stderr, &configPath, &profileOverride),
		newPushCmd(deps, stdout, stderr, &configPath, &profileOverride),
		newHelpCmd(rootCmd, stdout, stderr),
	)

	if err := rootCmd.Execute(); err != nil {
		var cmdErr *commandError
		if errors.As(err, &cmdErr) {
			if cmdErr.kind != commandErrorHelp {
				if _, writeErr := fmt.Fprintln(stderr, err.Error()); writeErr != nil {
					return 1
				}
			}
			return exitCodeForError(err)
		}
		if _, writeErr := fmt.Fprintln(stderr, err.Error()); writeErr != nil {
			return 1
		}
		return 2
	}

	// Cobra's built-in -h/--help returns nil from Execute() even if the
	// underlying write to stdout failed. Check the captured error.
	if helpWriteErr != nil {
		return exitCodeForError(outputError(helpWriteErr))
	}
	return 0
}

func newHelpCmd(rootCmd *cobra.Command, stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:                "help [command]",
		Short:              "Help about any command",
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Strip "-help" / "--help" that may leak through when
			// DisableFlagParsing is true.
			filtered := make([]string, 0, len(args))
			for _, a := range args {
				if a == "-help" || a == "--help" || a == "-h" {
					continue
				}
				filtered = append(filtered, a)
			}
			args = filtered

			if len(args) > 1 {
				return usageError(fmt.Errorf("help accepts at most one command name, got %d arguments", len(args)))
			}
			if len(args) == 0 {
				rootCmd.SetOut(stdout)
				if err := writeUsage(rootCmd, stdout); err != nil {
					return outputError(err)
				}
				return nil
			}
			target, _, err := rootCmd.Find(args)
			if err != nil || target == rootCmd || target.Name() == "help" {
				return usageError(fmt.Errorf("unknown command for help: %s", args[0]))
			}
			target.SetOut(stdout)
			if err := writeUsage(target, stdout); err != nil {
				return outputError(err)
			}
			return nil
		},
	}
}

// writeUsage renders the command's usage string to w, returning any write error.
// Cobra's cmd.Usage() delegates to a template that may silently swallow errors;
// this helper uses UsageString() + explicit write so the caller can handle I/O
// failures.
func writeUsage(cmd *cobra.Command, w io.Writer) error {
	usage := cmd.UsageString()
	_, err := strings.NewReader(usage).WriteTo(w)
	return err
}

func runtimeDepsMissing(deps Dependencies) bool {
	return deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil || deps.ResolveProjectPath == nil
}
