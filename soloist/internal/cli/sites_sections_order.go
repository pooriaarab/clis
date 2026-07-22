// Hand-authored (not generated). `sites sections move|hide` — reorder a section
// within a page, or remove it from pages (keeping it in websiteSettings.sections).

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSitesSectionsMoveCmd(flags *rootFlags) *cobra.Command {
	var page string
	var to int
	cmd := &cobra.Command{
		Use:     "move <siteId> <sectionId> --page /path --to N",
		Short:   "Reorder a section within a page (0-based position in that page).",
		Example: "  soloist-pp-cli sites sections move <siteId> <sectionId> --page / --to 2",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <sectionId>")
			}
			if !cmd.Flags().Changed("to") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--to N is required"))
			}
			if page == "" {
				page = "/"
			}
			siteID, sectionID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-move", "dry_run": true, "sectionId": sectionID, "page": page, "to": to}, "would reorder section")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			pg, ok := findPageByPath(state.Pages, page)
			if !ok {
				return usageErr(fmt.Errorf("page %q not found", page))
			}
			ids, _ := pg["sectionIds"].([]any)
			// remove current occurrence
			cur := -1
			for i, id := range ids {
				if s, _ := id.(string); s == sectionID {
					cur = i
					break
				}
			}
			if cur < 0 {
				return usageErr(fmt.Errorf("section %s is not on page %q", sectionID, page))
			}
			ids = append(ids[:cur], ids[cur+1:]...)
			if to < 0 {
				to = 0
			}
			if to > len(ids) {
				to = len(ids)
			}
			ids = append(ids[:to], append([]any{sectionID}, ids[to:]...)...)
			pg["sectionIds"] = ids
			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			resp, status, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(siteID), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-move", "status": status, "response": parseJSONForOutput(resp)}, "reordered section")
		},
	}
	cmd.Flags().StringVar(&page, "page", "/", "page path the section is on")
	cmd.Flags().IntVar(&to, "to", 0, "0-based target position within the page")
	return cmd
}

func newSitesSectionsHideCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hide <siteId> <sectionId>",
		Short:   "Remove a section from all pages (keeps it in the sections list).",
		Example: "  soloist-pp-cli sites sections hide <siteId> <sectionId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <sectionId>")
			}
			siteID, sectionID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-hide", "dry_run": true, "sectionId": sectionID}, "would hide section")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			removed := removeSectionIDFromPages(state.Pages, sectionID)
			if removed == 0 {
				return usageErr(fmt.Errorf("section %s is not on any page", sectionID))
			}
			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			resp, status, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(siteID), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-hide", "status": status, "removed_from_pages": removed, "response": parseJSONForOutput(resp)}, "hid section")
		},
	}
	return cmd
}
