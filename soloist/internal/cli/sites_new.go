// Hand-authored (not generated). `sites new` — create a new draft website, and
// a shared helper for calling the soloist.ai /api server routes with the stored
// bearer token (auto-refreshed).

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"soloist-pp-cli/internal/client"
	"soloist-pp-cli/internal/config"
)

const websitesCreatePath = "/api/websites/create"

// newSoloWebClient builds a client pointed at the soloist.ai web origin (for
// /api/* server routes), carrying the stored bearer token (refreshed if stale).
// A Referer header is included because some /api routes check the origin.
func newSoloWebClient(flags *rootFlags) (*client.Client, map[string]string, error) {
	cfg, err := config.Load(flags.configPath)
	if err != nil {
		return nil, nil, configErr(err)
	}
	maybeRefreshToken(cfg, flags.timeout)
	cfg.BaseURL = soloistWebBase()
	c := client.New(cfg, flags.timeout, flags.rateLimit)
	c.DryRun = flags.dryRun
	return c, map[string]string{
		"Referer":      soloistWebBase() + "/",
		"Content-Type": "application/json",
	}, nil
}

func newSitesNewCmd(flags *rootFlags) *cobra.Command {
	var from, handle, businessName string
	cmd := &cobra.Command{
		Use:   "new --from <siteId> [--handle H]",
		Short: "Create a new draft website (clone an existing one with --from).",
		Long: "Create a new draft website. --from <siteId> clones an existing site's " +
			"content into a fresh draft (guaranteed valid + publishable) — the reliable path. " +
			"Full from-scratch AI generation is the designer's multi-step onboarding; use " +
			"`sites ai` for individual generated content.",
		Example: "  soloist-pp-cli sites new --from 87c951ef-... --handle mynewsite",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--from <siteId> is required (clone an existing site); from-scratch AI generation is not yet wired"))
			}
			newID := uuid.NewString()

			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{
					"action": "sites-new", "dry_run": true, "from": from, "newId": newID, "handle": handle,
				}, fmt.Sprintf("would clone %s into a new draft %s", from, newID))
			}

			// Load the source draft's websiteSettings (Firestore-typed -> plain).
			fc, err := flags.newClient()
			if err != nil {
				return err
			}
			data, err := fc.Get(cmd.Context(), draftWebsitePath(from), map[string]string{})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var doc struct {
				Fields map[string]map[string]any `json:"fields"`
			}
			if err := json.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("parsing source draft: %w", err)
			}
			if doc.Fields == nil || doc.Fields["websiteSettings"] == nil {
				return fmt.Errorf("source draft %s has no websiteSettings", from)
			}
			ws, ok := fsDecode(doc.Fields["websiteSettings"]).(map[string]any)
			if !ok {
				return fmt.Errorf("websiteSettings malformed")
			}
			// Re-key the clone so it is a distinct site.
			ws["id"] = newID
			if handle != "" {
				ws["handle"] = handle
			}
			if businessName != "" {
				ws["businessName"] = businessName
			}

			// Create via the server route (writes DraftWebsites/<newID>).
			wc, hdr, err := newSoloWebClient(flags)
			if err != nil {
				return err
			}
			resp, status, err := wc.PostWithHeaders(cmd.Context(), websitesCreatePath, map[string]any{"websiteSettings": ws}, hdr)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if status < 200 || status >= 300 {
				return fmt.Errorf("create failed: HTTP %d: %s", status, string(resp))
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{
				"action": "sites-new", "status": status, "newId": newID, "clonedFrom": from,
			}, fmt.Sprintf("created draft %s (cloned from %s)", newID, from))
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "existing site id to clone")
	cmd.Flags().StringVar(&handle, "handle", "", "handle/slug for the new site")
	cmd.Flags().StringVar(&businessName, "business-name", "", "override the business name")
	return cmd
}
