package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bsmartlabs/dev-vault/internal/secretcontract"
	"github.com/bsmartlabs/dev-vault/internal/secretprovider"
	"github.com/bsmartlabs/dev-vault/internal/secretsync"
)

type listOptions struct {
	json         bool
	nameContains []string
	nameRegex    string
	path         string
	secretType   string
}

func newListCmd(deps Dependencies, stdout, stderr io.Writer, configPath, profileOverride *string) *cobra.Command {
	var opts listOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project -dev secrets metadata",
		Long: strings.Join([]string{
			"Lists secrets in the configured Scaleway project/region.",
			"This command always filters to secret names ending with '-dev'.",
			"It is not limited to entries present in .scw.json mapping.",
			"It never prints secret payloads, only metadata (name/type/path/id).",
		}, "\n"),
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError(fmt.Errorf("list does not accept positional arguments: %s", strings.Join(args, " ")))
			}
			return nil
		},
		Example: strings.Join([]string{
			"dev-vault list",
			"dev-vault list --json",
			"dev-vault list --name-contains bweb --name-contains env",
			"dev-vault list --name-regex '^bweb-env-.*-dev$' --path / --type key_value",
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtimeDepsMissing(deps) {
				return runtimeError(fmt.Errorf("internal error: missing dependencies"))
			}
			ctx := commandContext{stdout: stdout, stderr: stderr, deps: deps}
			params := commandParams{
				configPath:      *configPath,
				profileOverride: *profileOverride,
				configPolicy:    commandConfigProjectOnly,
			}
			return runListCmd(ctx, params, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output JSON")
	cmd.Flags().StringArrayVar(&opts.nameContains, "name-contains", nil, "Substring filter (repeatable, AND semantics)")
	cmd.Flags().StringVar(&opts.nameRegex, "name-regex", "", "Go regexp to match secret names")
	cmd.Flags().StringVar(&opts.path, "path", "", "Exact Scaleway secret path to filter")
	cmd.Flags().StringVar(&opts.secretType, "type", "", fmt.Sprintf("One of: %s", strings.Join(secretcontract.Names(), "|")))
	return cmd
}

func runListCmd(ctx commandContext, params commandParams, opts listOptions) error {
	query, err := buildListQuery(opts)
	if err != nil {
		return err
	}

	runtime := newCommandRuntime(ctx, params)
	resources, err := runtime.prepareResources(params.configPolicy)
	if err != nil {
		return err
	}

	filtered, err := resources.service.List(query)
	if err != nil {
		return runtimeError(err)
	}

	return renderListOutput(ctx, opts.json, filtered)
}

func buildListQuery(opts listOptions) (secretsync.ListQuery, error) {
	var re *regexp.Regexp
	var selectedType secretprovider.SecretType

	if opts.nameRegex != "" {
		compiled, err := regexp.Compile(opts.nameRegex)
		if err != nil {
			return secretsync.ListQuery{}, usageError(fmt.Errorf("invalid --name-regex: %w", err))
		}
		re = compiled
	}

	if opts.secretType != "" {
		parsedType, err := parseSecretType(opts.secretType)
		if err != nil {
			return secretsync.ListQuery{}, usageError(fmt.Errorf("invalid --type: %w", err))
		}
		selectedType = parsedType
	}

	return secretsync.ListQuery{
		NameContains: opts.nameContains,
		NameRegex:    re,
		Path:         opts.path,
		Type:         selectedType,
	}, nil
}

func renderListOutput(ctx commandContext, asJSON bool, filtered []secretsync.ListRecord) error {
	if asJSON {
		enc := json.NewEncoder(ctx.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(filtered); err != nil {
			return outputError(err)
		}
		return nil
	}

	if _, err := fmt.Fprintln(ctx.stdout, "NAME\tTYPE\tPATH\tID"); err != nil {
		return outputError(err)
	}
	for _, it := range filtered {
		if _, err := fmt.Fprintf(ctx.stdout, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID); err != nil {
			return outputError(err)
		}
	}
	return nil
}
