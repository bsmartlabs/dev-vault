package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/bsmartlabs/dev-vault/internal/mapping"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type pushOptions struct {
	all             bool
	yes             bool
	disablePrevious bool
	description     string
	createMissing   bool
}

func (o pushOptions) toServicePushOptions() secretsync.PushOptions {
	return secretsync.PushOptions{
		Description:     o.description,
		DisablePrevious: o.disablePrevious,
		CreateMissing:   o.createMissing,
	}
}

func newPushCmd(deps Dependencies, stdout, stderr io.Writer, configPath, profileOverride *string) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push (--all | <secret-dev> ...)",
		Short: "Push local files as new secret versions",
		Long: `Pushes one or more secrets from disk to Scaleway Secret Manager as a new version.
Secrets must exist in mapping and names must end with '-dev'.
Never prints secret payloads.

Formats:
  - mapping.format=raw reads file bytes as-is.
  - mapping.format=dotenv reads a .env file and uploads a JSON payload.

Notes:
  - --create-missing creates the secret if absent (requires mapping.type).
  - Secret creation uses mapping.path (default '/').
  - If more than one secret is being pushed, you must pass --yes.`,
		Args: cobra.ArbitraryArgs,
		Example: `dev-vault push bweb-env-bsmart-dev
dev-vault push bweb-env-bsmart-dev --description 'local refresh'
dev-vault push --all --yes
dev-vault push --config .scw.json --all --yes --disable-previous`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, params, err := newCommandInvocation(deps, stdout, stderr, configPath, profileOverride, commandConfigValidated, args)
			if err != nil {
				return err
			}
			return runPushBatch(ctx, params, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.all, "all", false, "Push all mapping entries with mode push")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm batch push (required when pushing more than one secret)")
	cmd.Flags().BoolVar(&opts.disablePrevious, "disable-previous", false, "Disable previous enabled version when creating a new version")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description for the new version (optional)")
	cmd.Flags().BoolVar(&opts.createMissing, "create-missing", false, "Create missing secrets (requires mapping.type)")
	return cmd
}

func reportPushBatchResults(ctx commandContext, result secretsync.PushBatchResult) error {
	return reportBatchResults(
		ctx,
		result.Succeeded,
		result.Failed,
		result.Summary,
		"push",
		func(item secretsync.PushResult) string {
			return fmt.Sprintf("pushed %s (rev=%d)", item.Name, item.Revision)
		},
	)
}

func runPushBatch(ctx commandContext, params commandParams, opts pushOptions) error {
	return runOperationBatch(
		ctx,
		params,
		mapping.ModePush,
		opts.all,
		func(targets []mapping.Target) error {
			if len(targets) > 1 && !opts.yes {
				return usageError(fmt.Errorf("refusing to push multiple secrets without --yes"))
			}
			return nil
		},
		func(service secretsync.Service, targets []mapping.Target) secretsync.PushBatchResult {
			return service.PushBatch(targets, opts.toServicePushOptions())
		},
		reportPushBatchResults,
	)
}
