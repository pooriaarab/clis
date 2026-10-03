package cli

import (
	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/check"
	"github.com/pooriaarab/clis/submit/internal/directories"
	"github.com/pooriaarab/clis/submit/internal/store"
)

func checkCmd(env *Env) *cobra.Command {
	var site, dir, listingURL string
	cmd := &cobra.Command{
		Use:   "check --site SITE --directory DIR",
		Short: "Verify a listing: fetch it, report status, follow vs nofollow, index hint",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			if site == "" || dir == "" {
				return usagef("check needs --site and --directory")
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
			siteURL := ""
			if listingURL == "" {
				rec := f.Find(site, entry.ID)
				if rec == nil {
					// --site may be a URL: match rows by stored site URL.
					for _, r := range f.FindBySiteURL(site) {
						if r.Directory == entry.ID {
							c := r
							rec = &c
							break
						}
					}
				}
				if rec == nil {
					return usagef("no tracker row for %s x %s; `track set` it first or pass --url", site, entry.ID)
				}
				if rec.ListingURL == "" {
					return usagef("row %s x %s has no listing URL; `track set` it with --listing-url or pass --url", site, entry.ID)
				}
				listingURL = rec.ListingURL
				siteURL = rec.SiteURL
			}
			if looksLikeURL(site) {
				siteURL = site
			}
			if siteURL == "" {
				return usagef("no site URL known for %s; pass the site URL as --site or store one with `track set --site-url`", site)
			}
			if env.DryRun {
				if env.P.JSON {
					return env.P.Object(map[string]any{"ok": true, "dry_run": true, "listing_url": listingURL, "site_url": siteURL})
				}
				env.P.Linef("would fetch %s", listingURL)
				env.P.Linef("dry run: nothing was fetched")
				return nil
			}
			res, err := check.Run(site, siteURL, entry.ID, listingURL)
			if err != nil {
				return err
			}
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "check": res})
			}
			env.P.Linef("listing: %s", res.ListingURL)
			env.P.Linef("status: %d", res.StatusCode)
			if len(res.Redirects) > 0 {
				env.P.Linef("redirects: %d -> %s", len(res.Redirects), res.FinalURL)
			}
			if res.LinkFound {
				env.P.Linef("link: found (%s) %s", res.LinkRel, res.LinkHref)
			} else {
				env.P.Linef("link: not found")
			}
			if res.PageNoFollow {
				env.P.Linef("page: meta robots nofollow")
			}
			env.P.Linef("hint: %s", res.IndexedHint)
			return nil
		}),
	}
	cmd.Flags().StringVar(&site, "site", "", "site id or site URL")
	cmd.Flags().StringVar(&dir, "directory", "", "directory id or name")
	cmd.Flags().StringVar(&listingURL, "url", "", "listing URL (default: from the tracker)")
	return cmd
}

func looksLikeURL(s string) bool {
	return len(s) > 8 && (hasPrefix(s, "http://") || hasPrefix(s, "https://"))
}

func hasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
