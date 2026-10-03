package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/config"
	"github.com/pooriaarab/clis/submit/internal/directories"
	"github.com/pooriaarab/clis/submit/internal/store"
)

func trackCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track",
		Short: "Track (site x directory) submission status in a local store",
	}
	cmd.AddCommand(trackInitCmd(env), trackSetCmd(env), trackListCmd(env))
	return cmd
}

func storePath(env *Env) (string, error) {
	dir, err := config.Dir(env.Getenv)
	if err != nil {
		return "", err
	}
	return config.StorePath(dir), nil
}

func trackInitCmd(env *Env) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the tracker store (no-op when it exists)",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			path, err := storePath(env)
			if err != nil {
				return err
			}
			if env.DryRun {
				if env.P.JSON {
					return env.P.Object(map[string]any{"ok": true, "dry_run": true, "store": path})
				}
				env.P.Linef("would init %s", path)
				env.P.Linef("dry run: nothing was written")
				return nil
			}
			f, err := store.Load(path)
			if err != nil {
				return err
			}
			if err := f.Save(path); err != nil {
				return err
			}
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "store": path})
			}
			env.P.Linef("store ready at %s", path)
			return nil
		}),
	}
}

func trackSetCmd(env *Env) *cobra.Command {
	var site, siteURL, dir, status, listingURL, date, note string
	cmd := &cobra.Command{
		Use:   "set --site ID --directory DIR --status STATUS",
		Short: "Set one (site x directory) row; refuses past 10 submissions/day",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			if site == "" || dir == "" || status == "" {
				return usagef("set needs --site, --directory and --status")
			}
			data, err := directories.Load()
			if err != nil {
				return err
			}
			entry := directories.ByName(data, dir)
			if entry == nil {
				return usagef("unknown directory %q (see `submit directories list`)", dir)
			}
			path, err := storePath(env)
			if err != nil {
				return err
			}
			f, err := store.Load(path)
			if err != nil {
				return err
			}
			rec, err := f.Set(store.SetInput{
				Site: site, SiteURL: siteURL, Directory: entry.ID,
				Status: status, ListingURL: listingURL, Date: date, Note: note,
				Now: time.Now(),
			})
			if err != nil {
				return usagef("%s", err)
			}
			if env.DryRun {
				if env.P.JSON {
					return env.P.Object(map[string]any{"ok": true, "dry_run": true, "record": rec})
				}
				env.P.Linef("would set %s x %s to %s", rec.Site, rec.Directory, rec.Status)
				env.P.Linef("dry run: nothing was written")
				return nil
			}
			if err := f.Save(path); err != nil {
				return err
			}
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "record": rec})
			}
			env.P.Linef("set %s x %s to %s", rec.Site, rec.Directory, rec.Status)
			return nil
		}),
	}
	cmd.Flags().StringVar(&site, "site", "", "site id (from the sites YAML)")
	cmd.Flags().StringVar(&siteURL, "site-url", "", "site URL (stored for `check`)")
	cmd.Flags().StringVar(&dir, "directory", "", "directory id or name (see `submit directories list`)")
	cmd.Flags().StringVar(&status, "status", "", "draft, submitted, live or rejected")
	cmd.Flags().StringVar(&listingURL, "listing-url", "", "listing page URL once known")
	cmd.Flags().StringVar(&date, "date", "", "activity date YYYY-MM-DD (default today)")
	cmd.Flags().StringVar(&note, "note", "", "free-text note (rejection reason, plan)")
	return cmd
}

func trackListCmd(env *Env) *cobra.Command {
	var site, dir, status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracker rows, optionally filtered",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			path, err := storePath(env)
			if err != nil {
				return err
			}
			f, err := store.Load(path)
			if err != nil {
				return err
			}
			rows := f.List(site, dir, status)
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "store": path, "records": rows})
			}
			if len(rows) == 0 {
				env.P.Linef("no rows")
				return nil
			}
			table := [][]string{{"SITE", "DIRECTORY", "STATUS", "DATE", "LISTING"}}
			for _, r := range rows {
				table = append(table, []string{r.Site, r.Directory, r.Status, r.Date, r.ListingURL})
			}
			env.P.Table(table)
			return nil
		}),
	}
	cmd.Flags().StringVar(&site, "site", "", "only this site id")
	cmd.Flags().StringVar(&dir, "directory", "", "only this directory id")
	cmd.Flags().StringVar(&status, "status", "", "only this status")
	return cmd
}
