package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type pullOptions struct {
	all       bool
	overwrite bool
}

func newPullCmd(deps Dependencies, stdout, stderr io.Writer, configPath, profileOverride *string) *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull (--all | <secret-dev> ...)",
		Short: "Pull mapped -dev secrets to local files",
		Long: `Pulls one or more secrets to disk based on .scw.json mapping.
Secrets must exist in mapping and names must end with '-dev'.
Pull reads the latest enabled secret version (Scaleway revision selector: latest_enabled).
Pull writes files atomically and chmods them to 0600 (on Unix).
Pull overwrites existing targets and creates missing parent directories.
Never prints secret payloads.

Formats:
  - mapping.format=raw writes secret bytes as-is.
  - mapping.format=dotenv expects a JSON object payload and renders deterministic .env output.`,
		Args: cobra.ArbitraryArgs,
		Example: `dev-vault pull bweb-env-bsmart-dev
dev-vault pull --all
dev-vault pull --config .scw.json bweb-env-bsmart-dev
dev-vault pull bweb-env-bsmart-dev --config .scw.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.overwrite = true
			ctx, params, err := newCommandInvocation(deps, stdout, stderr, configPath, profileOverride, commandConfigValidated, args)
			if err != nil {
				return err
			}
			return runPullBatch(ctx, params, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.all, "all", false, "Pull all mapping entries with mode pull")
	return cmd
}

func reportPullBatchResults(ctx commandContext, result secretsync.PullBatchResult) error {
	return reportBatchResults(
		ctx,
		result.Succeeded,
		result.Failed,
		result.Summary,
		"pull",
		func(item secretsync.PullResult) string {
			return fmt.Sprintf("pulled %s -> %s (rev=%d type=%s)", item.Name, item.File, item.Revision, item.Type)
		},
	)
}

func runPullBatch(ctx commandContext, params commandParams, opts pullOptions) error {
	return runOperationBatch(
		ctx,
		params,
		mapping.ModePull,
		opts.all,
		nil,
		func(service secretsync.Service, targets []mapping.Target) secretsync.PullBatchResult {
			return service.PullBatch(targets, opts.overwrite)
		},
		reportPullBatchResults,
	)
}
