package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const draftWebsiteDocumentPath = "/v1/projects/moz-ocho-solo-prod/databases/(default)/documents/DraftWebsites/{id}"
const draftWebsiteSectionsPagesPatchQuery = "?updateMask.fieldPaths=websiteSettings.sections&updateMask.fieldPaths=websiteSettings.pages"

type sitesHTTPClient interface {
	Get(ctx context.Context, path string, params map[string]string) (json.RawMessage, error)
	PatchWithParams(ctx context.Context, path string, params map[string]string, body any) (json.RawMessage, int, error)
}

type draftWebsiteState struct {
	Sections []map[string]any
	Pages    []map[string]any
}

type siteSectionRow struct {
	Index int    `json:"index"`
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

func newSitesSectionsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "sections",
		Short:       "List, add, and remove draft website sections.",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesSectionsListCmd(flags))
	cmd.AddCommand(newSitesSectionsAddCmd(flags))
	cmd.AddCommand(newSitesSectionsRmCmd(flags))
	cmd.AddCommand(newSitesSectionsEditCmd(flags))
	cmd.AddCommand(newSitesSectionsDupCmd(flags))
	cmd.AddCommand(newSitesSectionsMoveCmd(flags))
	cmd.AddCommand(newSitesSectionsHideCmd(flags))
	return cmd
}

func newSitesSectionsListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list <siteId>",
		Short:       "List content sections in a draft website.",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("too many arguments\nUsage: %s <siteId>", cmd.CommandPath()))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, args[0], flags)
			if err != nil {
				return err
			}
			rows := siteSectionRows(state.Sections)
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				tableRows := make([][]string, 0, len(rows))
				for _, row := range rows {
					tableRows = append(tableRows, []string{
						fmt.Sprint(row.Index),
						row.Type,
						row.ID,
						row.Title,
					})
				}
				return flags.printTable(cmd, []string{"INDEX", "TYPE", "ID", "TITLE"}, tableRows)
			}
			return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
		},
	}
	return cmd
}

func newSitesSectionsAddCmd(flags *rootFlags) *cobra.Command {
	var sectionType string
	var code string
	var title string
	var themeColor string
	var pagePath string

	cmd := &cobra.Command{
		Use:   "add <siteId>",
		Short: "Add a content section to a draft website and attach it to a page.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if len(args) > 1 {
				return usageErr(fmt.Errorf("too many arguments\nUsage: %s <siteId>", cmd.CommandPath()))
			}
			if strings.TrimSpace(sectionType) == "" {
				return usageErr(fmt.Errorf("type is required\nUsage: %s <siteId> --type <type>", cmd.CommandPath()))
			}

			section := buildSiteSection(uuid.NewString(), sectionType, 0, code, title, themeColor)
			sectionID, _ := section["id"].(string)
			if dryRunOK(flags) {
				previewPage := map[string]any{
					"path":       pagePath,
					"sectionIds": []any{sectionID},
				}
				patchBody := buildSitesSectionsPatchBody([]map[string]any{section}, []map[string]any{previewPage})
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run":     true,
					"action":      "add",
					"site_id":     args[0],
					"section":     section,
					"target_page": map[string]any{"path": pagePath},
					"patch_path":  draftWebsitePatchPath(args[0]),
					"patch_body":  patchBody,
					"note":        "dry-run does not fetch the current site; live mode patches the full current sections/pages arrays.",
				}, flags)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, args[0], flags)
			if err != nil {
				return err
			}
			section["typeOrdinal"] = countSectionsOfType(state.Sections, sectionType)
			page, ok := findPageByPath(state.Pages, pagePath)
			if !ok {
				return notFoundErr(fmt.Errorf("page path %q not found", pagePath))
			}
			appendSectionID(page, sectionID)
			state.Sections = append(state.Sections, section)

			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			data, statusCode, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(args[0]), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{
				"action":      "add",
				"site_id":     args[0],
				"section":     section,
				"target_page": map[string]any{"path": pagePath},
				"status":      statusCode,
				"success":     statusCode >= 200 && statusCode < 300,
				"response":    parseJSONForOutput(data),
			}, fmt.Sprintf("added section %s (%s) to %s", sectionID, sectionType, pagePath))
		},
	}
	cmd.Flags().StringVar(&sectionType, "type", "", "Section type to add, for example CodeEmbed")
	cmd.Flags().StringVar(&code, "code", "", "Raw HTML/script code for CodeEmbed sections")
	cmd.Flags().StringVar(&title, "title", "Code Embed", "Title for CodeEmbed sections")
	cmd.Flags().StringVar(&themeColor, "theme-color", "#0f172a", "Theme color for CodeEmbed sections")
	cmd.Flags().StringVar(&pagePath, "page", "/", "Page path to attach the section to")
	return cmd
}

func newSitesSectionsRmCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <siteId> <sectionId>",
		Short: "Remove a content section from a draft website and all pages.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return missingSitesArgument(cmd, flags, "<siteId> <sectionId>")
			}
			if len(args) == 1 {
				return usageErr(fmt.Errorf("sectionId is required\nUsage: %s <siteId> <sectionId>", cmd.CommandPath()))
			}
			if len(args) > 2 {
				return usageErr(fmt.Errorf("too many arguments\nUsage: %s <siteId> <sectionId>", cmd.CommandPath()))
			}
			siteID := args[0]
			sectionID := args[1]
			if dryRunOK(flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"dry_run":    true,
					"action":     "rm",
					"site_id":    siteID,
					"section_id": sectionID,
					"patch_path": draftWebsitePatchPath(siteID),
					"note":       "dry-run does not fetch the current site; live mode removes the section and strips it from every page.sectionIds array.",
				}, flags)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}
			state, err := loadDraftWebsiteState(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			index := findSectionIndex(state.Sections, sectionID)
			if index < 0 {
				return notFoundErr(fmt.Errorf("section %q not found", sectionID))
			}
			removed := state.Sections[index]
			state.Sections = append(state.Sections[:index], state.Sections[index+1:]...)
			pageReferenceCount := removeSectionIDFromPages(state.Pages, sectionID)

			body := buildSitesSectionsPatchBody(state.Sections, state.Pages)
			data, statusCode, err := c.PatchWithParams(cmd.Context(), draftWebsitePatchPath(siteID), nil, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{
				"action":                  "rm",
				"site_id":                 siteID,
				"section_id":              sectionID,
				"removed_section":         removed,
				"removed_page_references": pageReferenceCount,
				"status":                  statusCode,
				"success":                 statusCode >= 200 && statusCode < 300,
				"response":                parseJSONForOutput(data),
			}, fmt.Sprintf("removed section %s from %d page reference(s)", sectionID, pageReferenceCount))
		},
	}
	return cmd
}

func missingSitesArgument(cmd *cobra.Command, flags *rootFlags, usageSuffix string) error {
	if flags != nil && flags.asJSON {
		if err := printJSONFiltered(cmd.OutOrStdout(), map[string]any{
			"error": "missing required argument",
			"usage": fmt.Sprintf("%s %s", cmd.CommandPath(), usageSuffix),
		}, flags); err != nil {
			return err
		}
	}
	return usageErr(fmt.Errorf("missing required argument\nUsage: %s %s", cmd.CommandPath(), usageSuffix))
}

func loadDraftWebsiteState(ctx context.Context, c sitesHTTPClient, siteID string, flags *rootFlags) (*draftWebsiteState, error) {
	data, err := c.Get(ctx, draftWebsitePath(siteID), map[string]string{})
	if err != nil {
		return nil, classifyAPIError(err, flags)
	}
	return parseDraftWebsiteState(data)
}

func parseDraftWebsiteState(data json.RawMessage) (*draftWebsiteState, error) {
	var doc struct {
		Fields map[string]map[string]any `json:"fields"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing Firestore document: %w", err)
	}
	state := &draftWebsiteState{
		Sections: []map[string]any{},
		Pages:    []map[string]any{},
	}
	if doc.Fields == nil {
		return state, nil
	}
	websiteSettingsValue, ok := doc.Fields["websiteSettings"]
	if !ok {
		return state, nil
	}
	websiteSettings, ok := fsDecode(websiteSettingsValue).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("websiteSettings is not a Firestore map")
	}
	sections, err := mapArrayFromPlain(websiteSettings["sections"], "websiteSettings.sections")
	if err != nil {
		return nil, err
	}
	pages, err := mapArrayFromPlain(websiteSettings["pages"], "websiteSettings.pages")
	if err != nil {
		return nil, err
	}
	state.Sections = sections
	state.Pages = pages
	return state, nil
}

func mapArrayFromPlain(v any, name string) ([]map[string]any, error) {
	if v == nil {
		return []map[string]any{}, nil
	}
	values, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s is %T, want array", name, v)
	}
	out := make([]map[string]any, 0, len(values))
	for i, value := range values {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s[%d] is %T, want map", name, i, value)
		}
		out = append(out, m)
	}
	return out, nil
}

func draftWebsitePath(siteID string) string {
	return replacePathParam(draftWebsiteDocumentPath, "id", siteID)
}

func draftWebsitePatchPath(siteID string) string {
	return draftWebsitePath(siteID) + draftWebsiteSectionsPagesPatchQuery
}

func siteSectionRows(sections []map[string]any) []siteSectionRow {
	rows := make([]siteSectionRow, 0, len(sections))
	for i, section := range sections {
		props, _ := section["props"].(map[string]any)
		title, _ := props["title"].(string)
		rows = append(rows, siteSectionRow{
			Index: i,
			Type:  stringValue(section["type"]),
			ID:    stringValue(section["id"]),
			Title: title,
		})
	}
	return rows
}

func buildSiteSection(id, sectionType string, ordinal int, code, title, themeColor string) map[string]any {
	props := map[string]any{}
	if sectionType == "CodeEmbed" {
		props = map[string]any{
			"unsaved":    false,
			"code":       code,
			"title":      title,
			"themeColor": themeColor,
		}
	}
	return map[string]any{
		"id":          id,
		"type":        sectionType,
		"typeOrdinal": ordinal,
		"props":       props,
	}
}

func buildSitesSectionsPatchBody(sections, pages []map[string]any) map[string]any {
	return map[string]any{
		"fields": map[string]any{
			"websiteSettings": map[string]any{
				"mapValue": map[string]any{
					"fields": map[string]any{
						"sections": fsEncode(sections),
						"pages":    fsEncode(pages),
					},
				},
			},
		},
	}
}

func countSectionsOfType(sections []map[string]any, sectionType string) int {
	count := 0
	for _, section := range sections {
		if stringValue(section["type"]) == sectionType {
			count++
		}
	}
	return count
}

func findPageByPath(pages []map[string]any, path string) (map[string]any, bool) {
	for _, page := range pages {
		if stringValue(page["path"]) == path {
			return page, true
		}
	}
	return nil, false
}

func appendSectionID(page map[string]any, sectionID string) {
	ids, _ := page["sectionIds"].([]any)
	for _, existing := range ids {
		if stringValue(existing) == sectionID {
			return
		}
	}
	page["sectionIds"] = append(ids, sectionID)
}

func findSectionIndex(sections []map[string]any, sectionID string) int {
	for i, section := range sections {
		if stringValue(section["id"]) == sectionID {
			return i
		}
	}
	return -1
}

func removeSectionIDFromPages(pages []map[string]any, sectionID string) int {
	removed := 0
	for _, page := range pages {
		ids, ok := page["sectionIds"].([]any)
		if !ok {
			continue
		}
		next := ids[:0]
		for _, id := range ids {
			if stringValue(id) == sectionID {
				removed++
				continue
			}
			next = append(next, id)
		}
		page["sectionIds"] = next
	}
	return removed
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func parseJSONForOutput(data json.RawMessage) any {
	if len(data) == 0 {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return string(data)
	}
	return parsed
}

func writeSitesMutationResult(cmd *cobra.Command, flags *rootFlags, result map[string]any, human string) error {
	if !wantsHumanTable(cmd.OutOrStdout(), flags) {
		return printJSONFiltered(cmd.OutOrStdout(), result, flags)
	}
	fmt.Fprintln(cmd.OutOrStdout(), human)
	return nil
}
