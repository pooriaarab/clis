package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/config"
	"github.com/pooriaarab/clis/submit/internal/directories"
	"github.com/pooriaarab/clis/submit/internal/sites"
	"github.com/pooriaarab/clis/submit/internal/store"
)

func doctorCmd(env *Env) *cobra.Command {
	var flag string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and tracker store health",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			ok := true
			sitesPath := flag
			sitesSource := "flag --sites"
			if sitesPath == "" {
				sitesPath = env.Getenv("SUBMIT_SITES")
				sitesSource = "env SUBMIT_SITES"
			}
			if sitesPath == "" {
				sitesPath = "sites.yaml"
				sitesSource = "default ./sites.yaml"
			}
			sitesDetail := ""
			sitesN := 0
			list, err := sites.Load(sitesPath)
			switch {
			case err == nil:
				sitesN = len(list)
				sitesDetail = "ok"
			case errors.Is(err, os.ErrNotExist):
				ok = false
				sitesDetail = "not configured"
			default:
				ok = false
				sitesDetail = err.Error()
			}
			dir, derr := config.Dir(env.Getenv)
			storePath := ""
			if derr == nil {
				storePath = config.StorePath(dir)
			}
			f, serr := store.Load(storePath)
			storeDetail := "ok"
			byStatus := map[string]int{}
			if serr != nil {
				ok = false
				storeDetail = serr.Error()
			} else {
				for _, r := range f.Records {
					byStatus[r.Status]++
				}
				if _, err := os.Stat(storePath); os.IsNotExist(err) {
					storeDetail = "no store yet (run `track init`)"
				}
			}
			data, derr2 := directories.Load()
			dirsDetail := "ok"
			dirsN := 0
			if derr2 != nil {
				ok = false
				dirsDetail = derr2.Error()
			} else {
				dirsN = len(data.Directories)
			}
			if env.P.JSON {
				return env.P.Object(map[string]any{
					"ok": ok,
					"sites": map[string]any{
						"path": sitesPath, "source": sitesSource,
						"configured": sitesDetail == "ok", "count": sitesN, "detail": sitesDetail,
					},
					"store": map[string]any{
						"path": storePath, "detail": storeDetail, "by_status": byStatus,
					},
					"directories": map[string]any{"count": dirsN, "detail": dirsDetail},
				})
			}
			if sitesDetail == "ok" {
				env.P.Linef("sites: configured (%d sites, source: file %s)", sitesN, sitesPath)
			} else if sitesDetail == "not configured" {
				env.P.Linef("sites: not configured (no %s)", sitesPath)
			} else {
				env.P.Linef("sites: %s", sitesDetail)
			}
			env.P.Linef("store: %s (%s)", storeDetail, storePath)
			env.P.Linef("directories: %d built-in (%s)", dirsN, dirsDetail)
			return nil
		}),
	}
	cmd.Flags().StringVar(&flag, "sites", "", "sites YAML file (default $SUBMIT_SITES or ./sites.yaml)")
	return cmd
}
