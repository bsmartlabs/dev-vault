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
	fs.BoolVar(&all, "all", false, "Pull all mapping entries with mode pull|both (mode defaults to both)")
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
	fs.BoolVar(&all, "all", false, "Push all mapping entries with mode push|both (mode defaults to both)")
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
