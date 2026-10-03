package cli

import (
	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/sites"
)

// sitesFlag registers --sites on cmd and returns a loader. Resolution:
// --sites flag, SUBMIT_SITES env, ./sites.yaml.
func sitesFlag(env *Env, cmd *cobra.Command) func() ([]sites.Site, string, error) {
	var flag string
	cmd.Flags().StringVar(&flag, "sites", "", "sites YAML file (default $SUBMIT_SITES or ./sites.yaml)")
	return func() ([]sites.Site, string, error) {
		path := flag
		if path == "" {
			path = env.Getenv("SUBMIT_SITES")
		}
		if path == "" {
			path = "sites.yaml"
		}
		list, err := sites.Load(path)
		if err != nil {
			return nil, path, usagef("%s", err)
		}
		return list, path, nil
	}
}

// filterSites keeps only the ids in want (empty means all). Unknown ids are
// a usage error.
func filterSites(all []sites.Site, want []string) ([]sites.Site, error) {
	if len(want) == 0 {
		return all, nil
	}
	var out []sites.Site
	for _, id := range want {
		s := sites.ByID(all, id)
		if s == nil {
			return nil, usagef("unknown site id %q", id)
		}
		out = append(out, *s)
	}
	return out, nil
}
