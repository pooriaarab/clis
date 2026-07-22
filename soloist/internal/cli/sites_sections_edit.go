// Hand-authored (not generated). `sites sections edit|dup` — edit a section's
// props, or duplicate a section. Reuses helpers in sites_sections.go.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// deepCopyMap clones a plain map via a JSON round-trip.
func deepCopyMap(m map[string]any) (map[string]any, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func newSitesSectionsEditCmd(flags *rootFlags) *cobra.Command {
	var sets []string
	cmd := &cobra.Command{
		Use:     "edit <siteId> <sectionId> --set prop=value",
		Short:   "Edit a section's props (e.g. --set title=\"...\" --set code=\"<b>x</b>\").",
		Example: "  soloist-pp-cli sites sections edit <siteId> <sectionId> --set title=\"New heading\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <sectionId>")
			}
			if len(sets) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--set prop=value is required"))
			}
			updates, err := parseSetPairs(sets)
			if err != nil {
				return err
			}
			siteID, sectionID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-edit", "dry_run": true, "sectionId": sectionID, "props": updates}, "would edit section props")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			idx := findSectionIndex(state.Sections, sectionID)
			if idx < 0 {
				return usageErr(fmt.Errorf("section %s not found", sectionID))
			}
			props := asMap(state.Sections[idx]["props"])
			for k, v := range updates {
				props[k] = v
			}
			state.Sections[idx]["props"] = props
			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			resp, status, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(siteID), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-edit", "status": status, "sectionId": sectionID, "response": parseJSONForOutput(resp)}, "edited section")
		},
	}
	cmd.Flags().StringArrayVar(&sets, "set", nil, "prop=value (repeatable)")
	return cmd
}

func newSitesSectionsDupCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dup <siteId> <sectionId>",
		Short:   "Duplicate a section (inserted right after the original, on every page that shows it).",
		Example: "  soloist-pp-cli sites sections dup <siteId> <sectionId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <sectionId>")
			}
			siteID, sectionID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-dup", "dry_run": true, "sectionId": sectionID}, "would duplicate section")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			idx := findSectionIndex(state.Sections, sectionID)
			if idx < 0 {
				return usageErr(fmt.Errorf("section %s not found", sectionID))
			}
			clone, err := deepCopyMap(state.Sections[idx])
			if err != nil {
				return fmt.Errorf("copying section: %w", err)
			}
			newID := uuid.NewString()
			clone["id"] = newID
			// insert clone right after the original in the sections array
			newSections := make([]map[string]any, 0, len(state.Sections)+1)
			newSections = append(newSections, state.Sections[:idx+1]...)
			newSections = append(newSections, clone)
			newSections = append(newSections, state.Sections[idx+1:]...)
			state.Sections = newSections
			// on every page, insert newID right after sectionID in sectionIds
			for _, p := range state.Pages {
				ids, ok := p["sectionIds"].([]any)
				if !ok {
					continue
				}
				out := make([]any, 0, len(ids)+1)
				for _, id := range ids {
					out = append(out, id)
					if s, _ := id.(string); s == sectionID {
						out = append(out, newID)
					}
				}
				p["sectionIds"] = out
			}
			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			resp, status, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(siteID), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sections-dup", "status": status, "newSectionId": newID, "response": parseJSONForOutput(resp)}, fmt.Sprintf("duplicated section -> %s", newID))
		},
	}
	return cmd
}
