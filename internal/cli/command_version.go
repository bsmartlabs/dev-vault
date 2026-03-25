package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newVersionCmd(deps Dependencies, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version information",
		Long:  "Prints the build version/commit/date.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError(fmt.Errorf("version does not accept positional arguments: %s", strings.Join(args, " ")))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := fmt.Fprintf(stdout, "dev-vault %s (commit=%s date=%s)\n", deps.Version, deps.Commit, deps.Date); err != nil {
				return outputError(err)
			}
			return nil
		},
	}
}
