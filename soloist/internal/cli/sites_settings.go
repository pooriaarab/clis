// Hand-authored (not generated). Semantic edit commands over
// DraftWebsites/{id}.websiteSettings: theme, settings, pages. Reuses the
// fsDecode/fsEncode codec. See .printing-press-patches/sites-theme-settings-pages.md.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

// --- shared websiteSettings field helpers ---------------------------------

// loadWebsiteSettings GETs the draft doc and returns its websiteSettings as a
// plain (decoded) map.
func loadWebsiteSettings(ctx context.Context, c sitesHTTPClient, siteID string, flags *rootFlags) (map[string]any, error) {
	data, err := c.Get(ctx, draftWebsitePath(siteID), map[string]string{})
	if err != nil {
		return nil, classifyAPIError(err, flags)
	}
	var doc struct {
		Fields map[string]map[string]any `json:"fields"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing Firestore document: %w", err)
	}
	if doc.Fields == nil {
		return nil, fmt.Errorf("draft %s has no fields (not found, or not yours)", siteID)
	}
	wsVal, ok := doc.Fields["websiteSettings"]
	if !ok {
		return map[string]any{}, nil
	}
	ws, ok := fsDecode(wsVal).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("websiteSettings is not a map")
	}
	return ws, nil
}

// patchWebsiteSettingsFields PATCHes the given top-level websiteSettings
// sub-fields (plain values, encoded here) with an updateMask scoped to exactly
// those paths, leaving every other websiteSettings field untouched.
func patchWebsiteSettingsFields(ctx context.Context, c sitesHTTPClient, siteID string, fields map[string]any) (json.RawMessage, int, error) {
	encoded := map[string]any{}
	q := url.Values{}
	for k, v := range fields {
		encoded[k] = fsEncode(v)
		q.Add("updateMask.fieldPaths", "websiteSettings."+k)
	}
	body := map[string]any{
		"fields": map[string]any{
			"websiteSettings": map[string]any{
				"mapValue": map[string]any{"fields": encoded},
			},
		},
	}
	path := draftWebsitePath(siteID) + "?" + q.Encode()
	return c.PatchWithParams(ctx, path, nil, body)
}

// buildWSPatchPreview renders the dry-run body without sending.
func wsPatchPreview(siteID string, fields map[string]any) map[string]any {
	encoded := map[string]any{}
	paths := []string{}
	for k, v := range fields {
		encoded[k] = fsEncode(v)
		paths = append(paths, "websiteSettings."+k)
	}
	return map[string]any{
		"dry_run":    true,
		"patch_path": draftWebsitePath(siteID),
		"updateMask": paths,
		"fields":     encoded,
	}
}

// asMap coerces a decoded value to a map, defaulting to empty.
func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// parseSetPairs turns ["k=v","a=b"] into a map. Values are kept as strings.
func parseSetPairs(pairs []string) (map[string]any, error) {
	out := map[string]any{}
	for _, p := range pairs {
		i := strings.Index(p, "=")
		if i < 0 {
			return nil, usageErr(fmt.Errorf("--set expects key=value, got %q", p))
		}
		out[p[:i]] = p[i+1:]
	}
	return out, nil
}

// --- sites theme ----------------------------------------------------------

func newSitesThemeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "theme",
		Short:       "View and edit a draft website's theme (colors, fonts, corner/border styles).",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesThemeGetCmd(flags))
	cmd.AddCommand(newSitesThemeMergeCmd(flags, "colors", "colorScheme", "Merge color values into websiteSettings.colorScheme."))
	cmd.AddCommand(newSitesThemeMergeCmd(flags, "fonts", "fonts", "Merge font values into websiteSettings.fonts."))
	cmd.AddCommand(newSitesThemeStyleCmd(flags))
	return cmd
}

func newSitesThemeGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "get <siteId>",
		Short:   "Print the current theme (colorScheme, fonts, corner/border styles).",
		Example: "  soloist-pp-cli sites theme get <siteId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "theme-get", "dry_run": true}, "would read theme")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			ws, err := loadWebsiteSettings(cmd.Context(), c, args[0], flags)
			if err != nil {
				return err
			}
			out := map[string]any{}
			for _, k := range []string{"colorScheme", "fonts", "buttonCornerStyle", "imageCornerStyle", "bottomBorderStyle", "generationPresetStyle"} {
				if v, ok := ws[k]; ok {
					out[k] = v
				}
			}
			return writeSitesMutationResult(cmd, flags, out, "theme")
		},
	}
}

// newSitesThemeMergeCmd builds a `--set k=v` merge command for one map field.
func newSitesThemeMergeCmd(flags *rootFlags, name, field, short string) *cobra.Command {
	var sets []string
	cmd := &cobra.Command{
		Use:     name + " <siteId> --set key=value",
		Short:   short,
		Example: fmt.Sprintf("  soloist-pp-cli sites theme %s <siteId> --set primary=#4c1d95", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if len(sets) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--set key=value is required"))
			}
			updates, err := parseSetPairs(sets)
			if err != nil {
				return err
			}
			siteID := args[0]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, wsPatchPreview(siteID, map[string]any{field: updates}), "would merge "+field)
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			ws, err := loadWebsiteSettings(cmd.Context(), c, siteID, flags)
			if err != nil {
				return err
			}
			merged := asMap(ws[field])
			for k, v := range updates {
				merged[k] = v
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{field: merged})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": name, "status": status, "response": parseJSONForOutput(resp)}, fmt.Sprintf("updated %s", field))
		},
	}
	cmd.Flags().StringArrayVar(&sets, "set", nil, "key=value (repeatable)")
	return cmd
}

func newSitesThemeStyleCmd(flags *rootFlags) *cobra.Command {
	var button, image, border string
	cmd := &cobra.Command{
		Use:     "style <siteId> [--button-corner X] [--image-corner Y] [--bottom-border Z]",
		Short:   "Set corner/border styles.",
		Example: "  soloist-pp-cli sites theme style <siteId> --button-corner Rounded",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			fields := map[string]any{}
			if button != "" {
				fields["buttonCornerStyle"] = button
			}
			if image != "" {
				fields["imageCornerStyle"] = image
			}
			if border != "" {
				fields["bottomBorderStyle"] = border
			}
			if len(fields) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("set at least one of --button-corner / --image-corner / --bottom-border"))
			}
			siteID := args[0]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, wsPatchPreview(siteID, fields), "would set styles")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, fields)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "theme-style", "status": status, "response": parseJSONForOutput(resp)}, "updated styles")
		},
	}
	cmd.Flags().StringVar(&button, "button-corner", "", "buttonCornerStyle value")
	cmd.Flags().StringVar(&image, "image-corner", "", "imageCornerStyle value")
	cmd.Flags().StringVar(&border, "bottom-border", "", "bottomBorderStyle value")
	return cmd
}

// --- sites settings -------------------------------------------------------

func newSitesSettingsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "settings",
		Short:       "View and edit website settings (business details, language, GA4, SEO, cookie banner, head code, social).",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesSettingsGetCmd(flags))
	cmd.AddCommand(newSitesSettingsBusinessCmd(flags))
	cmd.AddCommand(newSitesSettingsScalarCmd(flags, "language", "language", "Set the website language."))
	cmd.AddCommand(newSitesSettingsScalarCmd(flags, "ga4", "gaMeasurementId", "Set the GA4 measurement id."))
	cmd.AddCommand(newSitesSettingsSeoCmd(flags))
	cmd.AddCommand(newSitesSettingsHeadCodeCmd(flags))
	cmd.AddCommand(newSitesSettingsSocialCmd(flags))
	return cmd
}

func newSitesSettingsGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "get <siteId>",
		Short:   "Print the current settings.",
		Example: "  soloist-pp-cli sites settings get <siteId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "settings-get", "dry_run": true}, "would read settings")
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			ws, err := loadWebsiteSettings(cmd.Context(), c, args[0], flags)
			if err != nil {
				return err
			}
			out := map[string]any{}
			for _, k := range []string{"businessName", "businessActivities", "businessLocation", "language", "gaMeasurementId", "indexForSearch", "cookieBanner", "codeContent", "socialMediaLinks", "AIOnboarding"} {
				if v, ok := ws[k]; ok {
					out[k] = v
				}
			}
			return writeSitesMutationResult(cmd, flags, out, "settings")
		},
	}
}

func newSitesSettingsBusinessCmd(flags *rootFlags) *cobra.Command {
	var name, activities, location string
	cmd := &cobra.Command{
		Use:     "business <siteId> [--name S] [--activities S] [--location S]",
		Short:   "Set business details.",
		Example: "  soloist-pp-cli sites settings business <siteId> --name \"Acme\" --activities \"Plumbing\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			fields := map[string]any{}
			if cmd.Flags().Changed("name") {
				fields["businessName"] = name
			}
			if cmd.Flags().Changed("activities") {
				fields["businessActivities"] = activities
			}
			if cmd.Flags().Changed("location") {
				fields["businessLocation"] = location
			}
			if len(fields) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("set at least one of --name / --activities / --location"))
			}
			return applyWSFields(cmd, flags, args[0], fields, "business")
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "businessName")
	cmd.Flags().StringVar(&activities, "activities", "", "businessActivities")
	cmd.Flags().StringVar(&location, "location", "", "businessLocation")
	return cmd
}

// newSitesSettingsScalarCmd sets one scalar websiteSettings string field from a positional.
func newSitesSettingsScalarCmd(flags *rootFlags, name, field, short string) *cobra.Command {
	return &cobra.Command{
		Use:     name + " <siteId> <value>",
		Short:   short,
		Example: fmt.Sprintf("  soloist-pp-cli sites settings %s <siteId> VALUE", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <value>")
			}
			return applyWSFields(cmd, flags, args[0], map[string]any{field: args[1]}, name)
		},
	}
}

func newSitesSettingsSeoCmd(flags *rootFlags) *cobra.Command {
	var index bool
	cmd := &cobra.Command{
		Use:     "seo <siteId> --index=true|false",
		Short:   "Set whether the site is indexed for search (indexForSearch).",
		Example: "  soloist-pp-cli sites settings seo <siteId> --index=true",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if !cmd.Flags().Changed("index") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--index=true|false is required"))
			}
			return applyWSFields(cmd, flags, args[0], map[string]any{"indexForSearch": index}, "seo")
		},
	}
	cmd.Flags().BoolVar(&index, "index", false, "index the site for search engines")
	return cmd
}

func newSitesSettingsHeadCodeCmd(flags *rootFlags) *cobra.Command {
	var code string
	var disable bool
	cmd := &cobra.Command{
		Use:     "head-code <siteId> --code '<script>...'",
		Short:   "Set custom head code (codeContent).",
		Example: "  soloist-pp-cli sites settings head-code <siteId> --code '<script>...</script>'",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if !cmd.Flags().Changed("code") && !disable {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--code is required (or --disable)"))
			}
			cc := map[string]any{"enabled": !disable, "code": code}
			return applyWSFields(cmd, flags, args[0], map[string]any{"codeContent": cc}, "head-code")
		},
	}
	cmd.Flags().StringVar(&code, "code", "", "raw head HTML/script")
	cmd.Flags().BoolVar(&disable, "disable", false, "disable custom head code")
	return cmd
}

func newSitesSettingsSocialCmd(flags *rootFlags) *cobra.Command {
	var sets []string
	cmd := &cobra.Command{
		Use:     "social <siteId> --set platform=url",
		Short:   "Set social media links (socialMediaLinks).",
		Example: "  soloist-pp-cli sites settings social <siteId> --set instagram=https://instagram.com/acme",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if len(sets) == 0 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--set platform=url is required"))
			}
			pairs, err := parseSetPairs(sets)
			if err != nil {
				return err
			}
			// socialMediaLinks is an array of {type, url}; build from the pairs.
			links := make([]any, 0, len(pairs))
			for platform, u := range pairs {
				links = append(links, map[string]any{"type": platform, "url": u})
			}
			return applyWSFields(cmd, flags, args[0], map[string]any{"socialMediaLinks": links}, "social")
		},
	}
	cmd.Flags().StringArrayVar(&sets, "set", nil, "platform=url (repeatable)")
	return cmd
}

// applyWSFields is the shared dry-run/patch tail for scalar/field setters.
func applyWSFields(cmd *cobra.Command, flags *rootFlags, siteID string, fields map[string]any, action string) error {
	if dryRunOK(flags) {
		return writeSitesMutationResult(cmd, flags, wsPatchPreview(siteID, fields), "would set "+action)
	}
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, fields)
	if err != nil {
		return classifyAPIError(err, flags)
	}
	return writeSitesMutationResult(cmd, flags, map[string]any{"action": action, "status": status, "response": parseJSONForOutput(resp)}, "updated "+action)
}
