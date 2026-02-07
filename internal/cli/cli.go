package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bsmartlabs/dev-vault/internal/config"
	"github.com/bsmartlabs/dev-vault/internal/dotenv"
	"github.com/bsmartlabs/dev-vault/internal/fsx"
	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

var getwdFn = os.Getwd

var allSecretTypes = []secret.SecretType{
	secret.SecretTypeOpaque,
	secret.SecretTypeCertificate,
	secret.SecretTypeKeyValue,
	secret.SecretTypeBasicCredentials,
	secret.SecretTypeDatabaseCredentials,
	secret.SecretTypeSSHKey,
}

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
}

func DefaultDependencies(version, commit, date string) Dependencies {
	return Dependencies{
		Version:       version,
		Commit:        commit,
		Date:          date,
		OpenSecretAPI: OpenScalewaySecretAPI,
		Now:           time.Now,
		Hostname:      os.Hostname,
	}
}

func OpenScalewaySecretAPI(cfg config.Config, profileOverride string) (SecretAPI, error) {
	profileName := strings.TrimSpace(profileOverride)
	if profileName == "" {
		profileName = strings.TrimSpace(cfg.Profile)
	}

	region, err := scw.ParseRegion(cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("invalid region %q: %w", cfg.Region, err)
	}

	var opts []scw.ClientOption
	if profileName != "" {
		scwCfg, err := scw.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("load scaleway config: %w", err)
		}
		prof, err := scwCfg.GetProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("get scaleway profile %q: %w", profileName, err)
		}
		opts = append(opts, scw.WithProfile(prof))
	}

	opts = append(opts,
		scw.WithEnv(),
		scw.WithDefaultOrganizationID(cfg.OrganizationID),
		scw.WithDefaultProjectID(cfg.ProjectID),
		scw.WithDefaultRegion(region),
	)

	client, err := scw.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("create scaleway client: %w", err)
	}

	return secret.NewAPI(client), nil
}

func Run(args []string, stdout, stderr io.Writer, deps Dependencies) int {
	if deps.OpenSecretAPI == nil || deps.Now == nil || deps.Hostname == nil {
		fmt.Fprintln(stderr, "internal error: missing dependencies")
		return 1
	}

	global := flag.NewFlagSet("dev-vault", flag.ContinueOnError)
	global.SetOutput(stderr)
	var configPath string
	var profileOverride string
	global.StringVar(&configPath, "config", "", "Path to .scw.json (default: search upward from cwd)")
	global.StringVar(&profileOverride, "profile", "", "Scaleway config profile override")

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

func printMainUsage(w io.Writer) {
	fmt.Fprintln(w, "dev-vault")
	fmt.Fprintln(w, "  Pull/push Scaleway Secret Manager secrets to disk for local development.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dev-vault [global options] <command> [command options] [args...]")
	fmt.Fprintln(w, "  dev-vault help [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global options:")
	fmt.Fprintf(w, "  --config <path>   Path to %s. If omitted: search upward from cwd.\n", config.DefaultConfigName)
	fmt.Fprintln(w, "  --profile <name>  Scaleway profile override (uses ~/.config/scw/config.yaml)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  version")
	fmt.Fprintln(w, "  list")
	fmt.Fprintln(w, "  pull")
	fmt.Fprintln(w, "  push")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Hard safety constraints:")
	fmt.Fprintln(w, "  - Refuses to operate on secret names that do not end with '-dev'.")
	fmt.Fprintln(w, "  - Never prints secret payloads.")
	fmt.Fprintln(w, "  - Pull writes files atomically and chmods them to 0600 (on Unix).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Batch behavior:")
	fmt.Fprintln(w, "  - pull --all includes mapping entries with mapping.mode in {pull, sync}.")
	fmt.Fprintln(w, "  - push --all includes mapping entries with mapping.mode in {push, sync}.")
	fmt.Fprintln(w, "  - If you do not use --all, you can ignore mapping.mode entirely.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  dev-vault list --json")
	fmt.Fprintln(w, "  dev-vault pull bweb-env-bsmart-dev --overwrite")
	fmt.Fprintln(w, "  dev-vault push bweb-env-bsmart-dev")
	fmt.Fprintln(w, "  dev-vault pull --config .scw.json bweb-env-bsmart-dev --overwrite")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Notes for automation/LLMs:")
	fmt.Fprintln(w, "  - Global options can be passed either before the command or as command options (e.g. 'pull --config ...').")
	fmt.Fprintln(w, "  - Exit codes: 0=success, 1=runtime error, 2=usage error.")
}

func printVersionUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dev-vault version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Prints the build version/commit/date.")
}

func printListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dev-vault [--config <path>] [--profile <name>] list [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Lists secrets in the configured Scaleway project/region.")
	fmt.Fprintln(w, "This command always filters to secret names ending with '-dev'.")
	fmt.Fprintln(w, "It never prints secret payloads, only metadata (name/type/path/id).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --config <path>         (optional) Path to .scw.json")
	fmt.Fprintln(w, "  --profile <name>        (optional) Scaleway profile override")
	fmt.Fprintln(w, "  --name-contains <s>     (repeatable) substring filter (AND semantics)")
	fmt.Fprintln(w, "  --name-regex <re>       Go regexp to match names")
	fmt.Fprintln(w, "  --path <p>              Exact Scaleway secret path to match (default: any)")
	fmt.Fprintln(w, "  --type <t>              One of: opaque|key_value|basic_credentials|database_credentials|ssh_key|certificate")
	fmt.Fprintln(w, "  --json                  Output JSON")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  dev-vault list")
	fmt.Fprintln(w, "  dev-vault list --json")
	fmt.Fprintln(w, "  dev-vault list --name-contains bweb --name-contains env")
	fmt.Fprintln(w, "  dev-vault list --name-regex '^bweb-env-.*-dev$' --path / --type key_value")
}

func printPullUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dev-vault [--config <path>] [--profile <name>] pull (--all | <secret-dev> ...) [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Pulls one or more secrets to disk based on .scw.json mapping.")
	fmt.Fprintln(w, "Secrets must exist in mapping and names must end with '-dev'.")
	fmt.Fprintln(w, "Pull reads the latest enabled secret version (Scaleway revision selector: latest_enabled).")
	fmt.Fprintln(w, "Pull writes files atomically and chmods them to 0600 (on Unix).")
	fmt.Fprintln(w, "Never prints secret payloads.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --config <path>   (optional) Path to .scw.json")
	fmt.Fprintln(w, "  --profile <name>  (optional) Scaleway profile override")
	fmt.Fprintln(w, "  --all             Pull all mapping entries with mapping.mode in {pull, sync}")
	fmt.Fprintln(w, "  --overwrite       Overwrite existing files (otherwise pull fails if the file exists)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Formats:")
	fmt.Fprintln(w, "  - mapping.format=raw")
	fmt.Fprintln(w, "    Writes secret bytes as-is.")
	fmt.Fprintln(w, "  - mapping.format=dotenv")
	fmt.Fprintln(w, "    Expects the secret payload to be a JSON object, renders a deterministic .env file:")
	fmt.Fprintln(w, "    - keys sorted lexicographically")
	fmt.Fprintln(w, "    - values quoted")
	fmt.Fprintln(w, "    - newlines and quotes escaped")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  dev-vault pull bweb-env-bsmart-dev --overwrite")
	fmt.Fprintln(w, "  dev-vault pull --all --overwrite")
	fmt.Fprintln(w, "  dev-vault pull --config .scw.json bweb-env-bsmart-dev --overwrite")
	fmt.Fprintln(w, "  dev-vault pull bweb-env-bsmart-dev --config .scw.json --overwrite")
}

func printPushUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dev-vault [--config <path>] [--profile <name>] push (--all | <secret-dev> ...) [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Pushes one or more secrets from disk to Scaleway Secret Manager as a new version.")
	fmt.Fprintln(w, "Secrets must exist in mapping and names must end with '-dev'.")
	fmt.Fprintln(w, "Never prints secret payloads.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --config <path>         (optional) Path to .scw.json")
	fmt.Fprintln(w, "  --profile <name>        (optional) Scaleway profile override")
	fmt.Fprintln(w, "  --all                   Push all mapping entries with mapping.mode in {push, sync}")
	fmt.Fprintln(w, "  --yes                   Required when pushing more than one secret (including --all)")
	fmt.Fprintln(w, "  --disable-previous      Disable the previously enabled version when creating the new version")
	fmt.Fprintln(w, "  --description <text>    Optional description for the new version (default is auto-generated)")
	fmt.Fprintln(w, "  --create-missing        Create the secret if it does not exist (requires mapping.type)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Formats:")
	fmt.Fprintln(w, "  - mapping.format=raw")
	fmt.Fprintln(w, "    Reads file bytes as-is and uploads them as a new secret version.")
	fmt.Fprintln(w, "  - mapping.format=dotenv")
	fmt.Fprintln(w, "    Reads a .env file and uploads a JSON object payload (key_value style).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  - --create-missing creates the secret if absent (requires mapping.type).")
	fmt.Fprintln(w, "  - Secret creation uses mapping.path (default is '/').")
	fmt.Fprintln(w, "  - If more than one secret is being pushed, you must pass --yes.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  dev-vault push bweb-env-bsmart-dev")
	fmt.Fprintln(w, "  dev-vault push bweb-env-bsmart-dev --description 'local refresh'")
	fmt.Fprintln(w, "  dev-vault push --all --yes")
	fmt.Fprintln(w, "  dev-vault push --config .scw.json --all --yes --disable-previous")
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runList(argv []string, stdout, stderr io.Writer, configPath, profileOverride string, deps Dependencies) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printListUsage(stderr) }

	cfgPath := configPath
	prof := profileOverride

	var contains stringSliceFlag
	var nameRegex string
	var pathFilter string
	var typeFilter string
	var jsonOut bool

	fs.StringVar(&cfgPath, "config", cfgPath, "Path to .scw.json (default: search upward from cwd)")
	fs.StringVar(&prof, "profile", prof, "Scaleway config profile override")
	fs.Var(&contains, "name-contains", "Substring filter (repeatable)")
	fs.StringVar(&nameRegex, "name-regex", "", "Go regexp to match secret names")
	fs.StringVar(&pathFilter, "path", "", "Exact Scaleway secret path to filter")
	fs.StringVar(&typeFilter, "type", "", "Secret type filter")
	fs.BoolVar(&jsonOut, "json", false, "Output JSON")

	argv = reorderFlags(argv, map[string]bool{
		"config":        true,
		"profile":       true,
		"name-contains": true,
		"name-regex":    true,
		"path":          true,
		"type":          true,
		"json":          false,
	})
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	loaded, api, ok := loadAndOpenAPI(cfgPath, prof, stderr, deps)
	if !ok {
		return 1
	}

	var re *regexp.Regexp
	if nameRegex != "" {
		compiled, err := regexp.Compile(nameRegex)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --name-regex: %v\n", err)
			return 2
		}
		re = compiled
	}

	var req secret.ListSecretsRequest
	req.Region = scw.Region(loaded.Cfg.Region)
	req.ProjectID = scw.StringPtr(loaded.Cfg.ProjectID)
	req.ScheduledForDeletion = false
	if pathFilter != "" {
		req.Path = scw.StringPtr(pathFilter)
	}
	types := allSecretTypes
	if typeFilter != "" {
		st, err := parseSecretType(typeFilter)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --type: %v\n", err)
			return 2
		}
		types = []secret.SecretType{st}
	}

	respSecrets, err := listSecretsByTypes(api, &req, types)
	if err != nil {
		fmt.Fprintf(stderr, "list secrets: %v\n", err)
		return 1
	}

	filtered := make([]listItem, 0, len(respSecrets))
	for _, s := range respSecrets {
		if s == nil {
			continue
		}
		if !strings.HasSuffix(s.Name, "-dev") {
			continue
		}
		if pathFilter != "" && s.Path != pathFilter {
			continue
		}
		if len(contains) > 0 {
			miss := false
			for _, c := range contains {
				if !strings.Contains(s.Name, c) {
					miss = true
					break
				}
			}
			if miss {
				continue
			}
		}
		if re != nil && !re.MatchString(s.Name) {
			continue
		}
		filtered = append(filtered, listItem{
			ID:   s.ID,
			Name: s.Name,
			Path: s.Path,
			Type: string(s.Type),
		})
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })

	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(filtered); err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
		return 0
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tPATH\tID")
	for _, it := range filtered {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", it.Name, it.Type, it.Path, it.ID)
	}
	_ = tw.Flush()
	return 0
}

type listItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

func runPull(argv []string, stdout, stderr io.Writer, configPath, profileOverride string, deps Dependencies) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printPullUsage(stderr) }

	cfgPath := configPath
	prof := profileOverride

	var all bool
	var overwrite bool
	fs.StringVar(&cfgPath, "config", cfgPath, "Path to .scw.json (default: search upward from cwd)")
	fs.StringVar(&prof, "profile", prof, "Scaleway config profile override")
	fs.BoolVar(&all, "all", false, "Pull all mapping entries with mode pull|sync")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	argv = reorderFlags(argv, map[string]bool{
		"config":    true,
		"profile":   true,
		"all":       false,
		"overwrite": false,
	})
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	loaded, api, ok := loadAndOpenAPI(cfgPath, prof, stderr, deps)
	if !ok {
		return 1
	}

	targets, err := selectMappingTargets(loaded.Cfg.Mapping, all, fs.Args(), "pull")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}

	for _, name := range targets {
		entry := loaded.Cfg.Mapping[name]
		outPath, err := config.ResolveFile(loaded.Root, entry.File)
		if err != nil {
			fmt.Fprintf(stderr, "mapping %s: resolve file: %v\n", name, err)
			return 1
		}

		s, err := resolveSecretByNameAndPath(api, loaded.Cfg, name, entry.Path)
		if err != nil {
			fmt.Fprintf(stderr, "resolve %s: %v\n", name, err)
			return 1
		}
		if entry.Type != "" && string(s.Type) != entry.Type {
			fmt.Fprintf(stderr, "secret %s: type mismatch (expected %s got %s)\n", name, entry.Type, s.Type)
			return 1
		}

		access, err := api.AccessSecretVersion(&secret.AccessSecretVersionRequest{
			Region:   scw.Region(loaded.Cfg.Region),
			SecretID: s.ID,
			Revision: "latest_enabled",
		})
		if err != nil {
			fmt.Fprintf(stderr, "access %s: %v\n", name, err)
			return 1
		}

		payload := access.Data
		if entry.Format == "dotenv" {
			converted, err := jsonToDotenv(payload)
			if err != nil {
				fmt.Fprintf(stderr, "format dotenv %s: %v\n", name, err)
				return 1
			}
			payload = converted
		}

		if err := fsx.AtomicWriteFile(outPath, payload, 0o600, overwrite); err != nil {
			if errors.Is(err, fsx.ErrExists) {
				fmt.Fprintf(stderr, "pull %s: file exists (use --overwrite): %s\n", name, outPath)
				return 1
			}
			fmt.Fprintf(stderr, "pull %s: write %s: %v\n", name, outPath, err)
			return 1
		}

		fmt.Fprintf(stdout, "pulled %s -> %s (rev=%d type=%s)\n", name, entry.File, access.Revision, access.Type)
	}

	return 0
}

func runPush(argv []string, stdout, stderr io.Writer, configPath, profileOverride string, deps Dependencies) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printPushUsage(stderr) }

	cfgPath := configPath
	prof := profileOverride

	var all bool
	var yes bool
	var disablePrevious bool
	var description string
	var createMissing bool

	fs.StringVar(&cfgPath, "config", cfgPath, "Path to .scw.json (default: search upward from cwd)")
	fs.StringVar(&prof, "profile", prof, "Scaleway config profile override")
	fs.BoolVar(&all, "all", false, "Push all mapping entries with mode push|sync")
	fs.BoolVar(&yes, "yes", false, "Confirm batch push (required when pushing more than one secret)")
	fs.BoolVar(&disablePrevious, "disable-previous", false, "Disable previous enabled version when creating a new version")
	fs.StringVar(&description, "description", "", "Description for the new version (optional)")
	fs.BoolVar(&createMissing, "create-missing", false, "Create missing secrets (requires mapping.type)")

	argv = reorderFlags(argv, map[string]bool{
		"config":           true,
		"profile":          true,
		"all":              false,
		"yes":              false,
		"disable-previous": false,
		"description":      true,
		"create-missing":   false,
	})
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	loaded, api, ok := loadAndOpenAPI(cfgPath, prof, stderr, deps)
	if !ok {
		return 1
	}

	targets, err := selectMappingTargets(loaded.Cfg.Mapping, all, fs.Args(), "push")
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 2
	}

	if len(targets) > 1 && !yes {
		fmt.Fprintln(stderr, "refusing to push multiple secrets without --yes")
		return 2
	}

	desc := description
	if desc == "" {
		host := "unknown-host"
		if h, err := deps.Hostname(); err == nil && h != "" {
			host = h
		}
		desc = fmt.Sprintf("dev-vault push %s %s", deps.Now().UTC().Format(time.RFC3339), host)
	}

	for _, name := range targets {
		entry := loaded.Cfg.Mapping[name]
		inPath, err := config.ResolveFile(loaded.Root, entry.File)
		if err != nil {
			fmt.Fprintf(stderr, "mapping %s: resolve file: %v\n", name, err)
			return 1
		}
		raw, err := os.ReadFile(inPath)
		if err != nil {
			fmt.Fprintf(stderr, "push %s: read %s: %v\n", name, inPath, err)
			return 1
		}

		payload := raw
		if entry.Format == "dotenv" {
			converted, err := dotenvToJSON(payload)
			if err != nil {
				fmt.Fprintf(stderr, "format dotenv %s: %v\n", name, err)
				return 1
			}
			payload = converted
		}

		s, err := resolveSecretByNameAndPath(api, loaded.Cfg, name, entry.Path)
		if err != nil {
			var nf *notFoundError
			if errors.As(err, &nf) && createMissing {
				if entry.Type == "" {
					fmt.Fprintf(stderr, "push %s: create-missing requires mapping.type\n", name)
					return 1
				}
				st, err := parseSecretType(entry.Type)
				if err != nil {
					fmt.Fprintf(stderr, "push %s: invalid mapping.type: %v\n", name, err)
					return 1
				}
				_, err = api.CreateSecret(&secret.CreateSecretRequest{
					Region:      scw.Region(loaded.Cfg.Region),
					ProjectID:   loaded.Cfg.ProjectID,
					Name:        name,
					Tags:        []string{},
					Description: nil,
					Type:        st,
					Path:        scw.StringPtr(entry.Path),
					Protected:   false,
					KeyID:       nil,
				})
				if err != nil {
					fmt.Fprintf(stderr, "push %s: create secret: %v\n", name, err)
					return 1
				}
				s, err = resolveSecretByNameAndPath(api, loaded.Cfg, name, entry.Path)
				if err != nil {
					fmt.Fprintf(stderr, "push %s: after create, resolve failed: %v\n", name, err)
					return 1
				}
			} else {
				fmt.Fprintf(stderr, "resolve %s: %v\n", name, err)
				return 1
			}
		}

		if entry.Type != "" && string(s.Type) != entry.Type {
			fmt.Fprintf(stderr, "secret %s: type mismatch (expected %s got %s)\n", name, entry.Type, s.Type)
			return 1
		}

		disablePrevPtr := (*bool)(nil)
		if disablePrevious {
			disablePrevPtr = scw.BoolPtr(true)
		}
		descPtr := scw.StringPtr(desc)

		v, err := api.CreateSecretVersion(&secret.CreateSecretVersionRequest{
			Region:          scw.Region(loaded.Cfg.Region),
			SecretID:        s.ID,
			Data:            payload,
			Description:     descPtr,
			DisablePrevious: disablePrevPtr,
		})
		if err != nil {
			fmt.Fprintf(stderr, "push %s: create version: %v\n", name, err)
			return 1
		}
		fmt.Fprintf(stdout, "pushed %s (rev=%d)\n", name, v.Revision)
	}

	return 0
}

func reorderFlags(argv []string, takesValue map[string]bool) []string {
	// Go's standard flag package stops parsing when it sees the first non-flag argument.
	// For a better CLI UX, accept flags after positional args by reordering them.
	var flags []string
	var positional []string

	normalize := func(tok string) string {
		tok = strings.TrimLeft(tok, "-")
		if i := strings.IndexByte(tok, '='); i >= 0 {
			tok = tok[:i]
		}
		return tok
	}

	for i := 0; i < len(argv); i++ {
		tok := argv[i]
		if tok == "--" {
			positional = append(positional, argv[i+1:]...)
			break
		}
		if strings.HasPrefix(tok, "-") && tok != "-" {
			flags = append(flags, tok)
			name := normalize(tok)
			if takesValue[name] && !strings.Contains(tok, "=") && i+1 < len(argv) {
				flags = append(flags, argv[i+1])
				i++
			}
			continue
		}
		positional = append(positional, tok)
	}

	return append(flags, positional...)
}

func loadAndOpenAPI(configPath, profileOverride string, stderr io.Writer, deps Dependencies) (*config.Loaded, SecretAPI, bool) {
	wd, err := getwdFn()
	if err != nil {
		fmt.Fprintf(stderr, "getwd: %v\n", err)
		return nil, nil, false
	}
	loaded, err := config.Load(wd, configPath)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return nil, nil, false
	}
	api, err := deps.OpenSecretAPI(loaded.Cfg, profileOverride)
	if err != nil {
		fmt.Fprintf(stderr, "open scaleway api: %v\n", err)
		return nil, nil, false
	}
	return loaded, api, true
}

type notFoundError struct {
	name string
	path string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("secret not found: name=%s path=%s", e.name, e.path)
}

func resolveSecretByNameAndPath(api SecretAPI, cfg config.Config, name, path string) (*secret.Secret, error) {
	req := secret.ListSecretsRequest{
		Region:               scw.Region(cfg.Region),
		ProjectID:            scw.StringPtr(cfg.ProjectID),
		Name:                 scw.StringPtr(name),
		Path:                 scw.StringPtr(path),
		ScheduledForDeletion: false,
	}
	respSecrets, err := listSecretsByTypes(api, &req, allSecretTypes)
	if err != nil {
		return nil, err
	}

	matches := make([]*secret.Secret, 0, len(respSecrets))
	for _, s := range respSecrets {
		if s != nil && s.Name == name && s.Path == path {
			matches = append(matches, s)
		}
	}
	if len(matches) == 0 {
		return nil, &notFoundError{name: name, path: path}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, s := range matches {
			ids = append(ids, s.ID)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("multiple secrets match name=%s path=%s: %s", name, path, strings.Join(ids, ","))
	}
	return matches[0], nil
}

func listSecretsByTypes(api SecretAPI, base *secret.ListSecretsRequest, types []secret.SecretType) ([]*secret.Secret, error) {
	var out []*secret.Secret
	for _, t := range types {
		req := *base
		req.Type = t
		resp, err := api.ListSecrets(&req, scw.WithAllPages())
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Secrets...)
	}
	return out, nil
}

func selectMappingTargets(mapping map[string]config.MappingEntry, all bool, positional []string, mode string) ([]string, error) {
	if all && len(positional) > 0 {
		return nil, errors.New("cannot use --all with explicit secret names")
	}
	if !all && len(positional) == 0 {
		return nil, errors.New("no secrets specified (use --all or pass secret names)")
	}

	isAllowedMode := func(entry config.MappingEntry) bool {
		switch mode {
		case "pull":
			return entry.Mode == "pull" || entry.Mode == "sync"
		case "push":
			return entry.Mode == "push" || entry.Mode == "sync"
		default:
			return false
		}
	}

	var out []string
	if all {
		for name, entry := range mapping {
			if isAllowedMode(entry) {
				out = append(out, name)
			}
		}
		sort.Strings(out)
		if len(out) == 0 {
			return nil, fmt.Errorf("no mapping entries selected for %s", mode)
		}
		return out, nil
	}

	for _, name := range positional {
		if !strings.HasSuffix(name, "-dev") {
			return nil, fmt.Errorf("refusing non-dev secret name: %s", name)
		}
		entry, ok := mapping[name]
		if !ok {
			return nil, fmt.Errorf("secret not found in mapping: %s", name)
		}
		if !isAllowedMode(entry) {
			return nil, fmt.Errorf("secret %s not allowed in %s mode (mapping.mode=%s)", name, mode, entry.Mode)
		}
		out = append(out, name)
	}
	return out, nil
}

func parseSecretType(s string) (secret.SecretType, error) {
	switch s {
	case "opaque":
		return secret.SecretTypeOpaque, nil
	case "certificate":
		return secret.SecretTypeCertificate, nil
	case "key_value":
		return secret.SecretTypeKeyValue, nil
	case "basic_credentials":
		return secret.SecretTypeBasicCredentials, nil
	case "database_credentials":
		return secret.SecretTypeDatabaseCredentials, nil
	case "ssh_key":
		return secret.SecretTypeSSHKey, nil
	default:
		return "", fmt.Errorf("unknown secret type %q", s)
	}
}

func jsonToDotenv(payload []byte) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	env := make(map[string]string, len(m))
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			env[k] = vv
		default:
			// Values come from json.Unmarshal into interface{}, so they are always JSON-marshalable.
			env[k] = string(mustJSONMarshal(v))
		}
	}
	return dotenv.Render(env), nil
}

func mustJSONMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func dotenvToJSON(payload []byte) ([]byte, error) {
	env, err := dotenv.Parse(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(env)
}
