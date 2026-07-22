// Hand-authored (not generated). `sites ai` — wrapper over the Solo
// /api/ai content-generation route (the building block the designer's website
// generation orchestrates). Contract captured from the live deployed API:
// POST /api/ai {docId, prompt, schema, genLanguage} -> {result} where result is
// a JSON string when a schema is given. schema values are lowercase short names
// (intro, services, unsplash, quotes, about, faq, reviews, team, contact,
// colors, fonts, video, scheduling, newsletter, blog, ...). genLanguage is a
// lowercase language name (default "english").

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const aiGeneratePath = "/api/ai"

func newSitesAICmd(flags *rootFlags) *cobra.Command {
	var prompt, schema, language, docID string
	cmd := &cobra.Command{
		Use:   "ai --prompt \"...\" [--schema intro] [--doc-id <id>]",
		Short: "Generate content with Solo's AI (POST /api/ai).",
		Long: "Calls the Solo AI route with a prompt and optional response schema, printing the result. " +
			"Schema values are lowercase short names mirroring the designer's generators: " +
			"intro, services, unsplash, quotes, about, faq, reviews, team, contact, colors, fonts, " +
			"video, scheduling, newsletter, blog. Omit --schema for freeform text.",
		Example: "  soloist-pp-cli sites ai --prompt \"A cozy Boston dry cleaner named Sudsy\" --schema intro",
		RunE: func(cmd *cobra.Command, args []string) error {
			if prompt == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--prompt is required"))
			}
			if docID == "" {
				docID = uuid.NewString()
			}
			if language == "" {
				language = "english"
			}
			body := map[string]any{"docId": docID, "prompt": prompt, "genLanguage": language}
			if schema != "" {
				body["schema"] = schema
			}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "sites-ai", "dry_run": true, "request": body}, "would call /api/ai")
			}
			wc, hdr, err := newSoloWebClient(flags)
			if err != nil {
				return err
			}
			resp, status, err := wc.PostWithHeaders(cmd.Context(), aiGeneratePath, body, hdr)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if status < 200 || status >= 300 {
				return fmt.Errorf("ai generation failed: HTTP %d: %s", status, string(resp))
			}
			var parsed struct {
				Result string `json:"result"`
			}
			_ = json.Unmarshal(resp, &parsed)
			out := map[string]any{"action": "sites-ai", "status": status}
			if parsed.Result != "" {
				var inner any
				if json.Unmarshal([]byte(parsed.Result), &inner) == nil {
					out["result"] = inner // schema mode -> structured JSON
				} else {
					out["result"] = parsed.Result // freeform text
				}
			} else {
				out["result"] = parseJSONForOutput(resp)
			}
			return writeSitesMutationResult(cmd, flags, out, "generated")
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "the generation prompt (required)")
	cmd.Flags().StringVar(&schema, "schema", "", "response schema (lowercase: intro, services, quotes, about, faq, reviews, ...)")
	cmd.Flags().StringVar(&language, "language", "", "generation language (lowercase, default english)")
	cmd.Flags().StringVar(&docID, "doc-id", "", "doc id for logging/context (default: random uuid)")
	return cmd
}
