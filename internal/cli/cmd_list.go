package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	secret "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

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

	bindGlobalOptionFlags(fs, &cfgPath, &prof)
	fs.Var(&contains, "name-contains", "Substring filter (repeatable)")
	fs.StringVar(&nameRegex, "name-regex", "", "Go regexp to match secret names")
	fs.StringVar(&pathFilter, "path", "", "Exact Scaleway secret path to filter")
	fs.StringVar(&typeFilter, "type", "", "Secret type filter")
	fs.BoolVar(&jsonOut, "json", false, "Output JSON")

	argv = reorderFlags(argv, withGlobalFlagSpecs(map[string]bool{
		"name-contains": true,
		"name-regex":    true,
		"path":          true,
		"type":          true,
		"json":          false,
	}))
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	runtime, err := buildCommandRuntime(cfgPath, prof, deps)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
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
	req.Region = scw.Region(runtime.loaded.Cfg.Region)
	req.ProjectID = scw.StringPtr(runtime.loaded.Cfg.ProjectID)
	req.ScheduledForDeletion = false
	if pathFilter != "" {
		req.Path = scw.StringPtr(pathFilter)
	}
	types := supportedSecretTypes()
	if typeFilter != "" {
		st, err := parseSecretType(typeFilter)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --type: %v\n", err)
			return 2
		}
		types = []secret.SecretType{st}
	}

	respSecrets, err := listSecretsByTypes(runtime.api, &req, types)
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
