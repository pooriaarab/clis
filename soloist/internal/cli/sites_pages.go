// Hand-authored (not generated). `sites pages` — CRUD over
// DraftWebsites/{id}.websiteSettings.pages. See the patch note.

package cli

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newSitesPagesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "pages",
		Short:       "List and edit a draft website's pages.",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesPagesListCmd(flags))
	cmd.AddCommand(newSitesPagesCreateCmd(flags))
	cmd.AddCommand(newSitesPagesRmCmd(flags))
	cmd.AddCommand(newSitesPagesSetCmd(flags))
	return cmd
}

// loadPages returns the decoded websiteSettings plus its pages as a []map.
func loadPages(cmd *cobra.Command, flags *rootFlags, siteID string) (map[string]any, []map[string]any, error) {
	c, err := flags.newClient()
	if err != nil {
		return nil, nil, err
	}
	ws, err := loadWebsiteSettings(cmd.Context(), c, siteID, flags)
	if err != nil {
		return nil, nil, err
	}
	pages, err := mapArrayFromPlain(ws["pages"], "websiteSettings.pages")
	if err != nil {
		return nil, nil, err
	}
	return ws, pages, nil
}

func newSitesPagesListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "list <siteId>",
		Short:   "List pages (id, title, path, section count).",
		Example: "  soloist-pp-cli sites pages list <siteId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-list", "dry_run": true}, "would list pages")
			}
			_, pages, err := loadPages(cmd, flags, args[0])
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(pages))
			for i, p := range pages {
				sectionIds, _ := p["sectionIds"].([]any)
				rows = append(rows, map[string]any{
					"index":    i,
					"id":       stringValue(p["id"]),
					"title":    stringValue(p["title"]),
					"path":     stringValue(p["path"]),
					"sections": len(sectionIds),
				})
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"pages": rows}, fmt.Sprintf("%d pages", len(rows)))
		},
	}
}

func splitKeywords(s string) []any {
	if strings.TrimSpace(s) == "" {
		return []any{}
	}
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func newSitesPagesCreateCmd(flags *rootFlags) *cobra.Command {
	var title, path, keywords string
	cmd := &cobra.Command{
		Use:     "create <siteId> --title S --path /p [--keywords k1,k2]",
		Short:   "Add a page.",
		Example: "  soloist-pp-cli sites pages create <siteId> --title About --path /about",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if title == "" || path == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--title and --path are required"))
			}
			page := map[string]any{
				"id":         uuid.NewString(),
				"title":      title,
				"path":       path,
				"keywords":   splitKeywords(keywords),
				"sectionIds": []any{},
			}
			siteID := args[0]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-create", "dry_run": true, "page": page}, "would create page")
			}
			_, pages, err := loadPages(cmd, flags, siteID)
			if err != nil {
				return err
			}
			newPages := make([]any, 0, len(pages)+1)
			for _, p := range pages {
				newPages = append(newPages, p)
			}
			newPages = append(newPages, page)
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{"pages": newPages})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-create", "status": status, "pageId": page["id"], "response": parseJSONForOutput(resp)}, fmt.Sprintf("created page %s", path))
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "page title")
	cmd.Flags().StringVar(&path, "path", "", "page path, e.g. /about")
	cmd.Flags().StringVar(&keywords, "keywords", "", "comma-separated SEO keywords")
	return cmd
}

func newSitesPagesRmCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <siteId> <pageId>",
		Short:   "Remove a page (its sections are left in websiteSettings.sections).",
		Example: "  soloist-pp-cli sites pages rm <siteId> <pageId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <pageId>")
			}
			siteID, pageID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-rm", "dry_run": true, "pageId": pageID}, "would remove page")
			}
			_, pages, err := loadPages(cmd, flags, siteID)
			if err != nil {
				return err
			}
			kept := make([]any, 0, len(pages))
			found := false
			for _, p := range pages {
				if stringValue(p["id"]) == pageID {
					found = true
					continue
				}
				kept = append(kept, p)
			}
			if !found {
				return usageErr(fmt.Errorf("page %s not found", pageID))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{"pages": kept})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-rm", "status": status, "response": parseJSONForOutput(resp)}, "removed page")
		},
	}
}

func newSitesPagesSetCmd(flags *rootFlags) *cobra.Command {
	var title, path, keywords string
	cmd := &cobra.Command{
		Use:     "set <siteId> <pageId> [--title S] [--path P] [--keywords k1,k2]",
		Short:   "Edit a page's title / path / keywords.",
		Example: "  soloist-pp-cli sites pages set <siteId> <pageId> --title \"New Title\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <pageId>")
			}
			if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("path") && !cmd.Flags().Changed("keywords") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("set at least one of --title / --path / --keywords"))
			}
			siteID, pageID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-set", "dry_run": true, "pageId": pageID}, "would edit page")
			}
			_, pages, err := loadPages(cmd, flags, siteID)
			if err != nil {
				return err
			}
			found := false
			out := make([]any, 0, len(pages))
			for _, p := range pages {
				if stringValue(p["id"]) == pageID {
					found = true
					if cmd.Flags().Changed("title") {
						p["title"] = title
					}
					if cmd.Flags().Changed("path") {
						p["path"] = path
					}
					if cmd.Flags().Changed("keywords") {
						p["keywords"] = splitKeywords(keywords)
					}
				}
				out = append(out, p)
			}
			if !found {
				return usageErr(fmt.Errorf("page %s not found", pageID))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{"pages": out})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "pages-set", "status": status, "response": parseJSONForOutput(resp)}, "updated page")
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "page title")
	cmd.Flags().StringVar(&path, "path", "", "page path")
	cmd.Flags().StringVar(&keywords, "keywords", "", "comma-separated SEO keywords")
	return cmd
}
