// Hand-authored (not generated). `sites blog` — list/add/rm blog posts stored
// in DraftWebsites/{id}.websiteSettings.pregeneratedBlogPosts.

package cli

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const blogPostsField = "pregeneratedBlogPosts"

func newSitesBlogCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "blog",
		Short:       "List, add, and remove blog posts.",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesBlogListCmd(flags))
	cmd.AddCommand(newSitesBlogAddCmd(flags))
	cmd.AddCommand(newSitesBlogRmCmd(flags))
	return cmd
}

func loadBlogPosts(cmd *cobra.Command, flags *rootFlags, siteID string) ([]map[string]any, error) {
	c, err := flags.newClient()
	if err != nil {
		return nil, err
	}
	ws, err := loadWebsiteSettings(cmd.Context(), c, siteID, flags)
	if err != nil {
		return nil, err
	}
	return mapArrayFromPlain(ws[blogPostsField], "websiteSettings."+blogPostsField)
}

func newSitesBlogListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "list <siteId>",
		Short:   "List blog posts (id, title, date).",
		Example: "  soloist-pp-cli sites blog list <siteId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "blog-list", "dry_run": true}, "would list blog posts")
			}
			posts, err := loadBlogPosts(cmd, flags, args[0])
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(posts))
			for i, p := range posts {
				rows = append(rows, map[string]any{
					"index":  i,
					"id":     stringValue(p["id"]),
					"title":  stringValue(p["title"]),
					"date":   stringValue(p["date"]),
					"author": stringValue(p["authorName"]),
				})
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"posts": rows}, fmt.Sprintf("%d blog posts", len(rows)))
		},
	}
}

func newSitesBlogAddCmd(flags *rootFlags) *cobra.Command {
	var title, body, author, date, imageURL, imageKeywords string
	cmd := &cobra.Command{
		Use:     "add <siteId> --title S --body '<p>...</p>'",
		Short:   "Add a blog post.",
		Example: "  soloist-pp-cli sites blog add <siteId> --title \"Hello\" --body \"<p>First post</p>\" --author \"Jane\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 1 {
				return missingSitesArgument(cmd, flags, "<siteId>")
			}
			if title == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--title is required"))
			}
			if date == "" {
				date = time.Now().UTC().Format("2006-01-02")
			}
			image := map[string]any{}
			if imageURL != "" {
				image["url"] = imageURL
			}
			post := map[string]any{
				"id":            uuid.NewString(),
				"title":         title,
				"body":          body,
				"image":         image,
				"imageKeywords": imageKeywords,
				"authorName":    author,
				"date":          date,
				"isAIAssisted":  false,
			}
			siteID := args[0]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "blog-add", "dry_run": true, "post": post}, "would add blog post")
			}
			posts, err := loadBlogPosts(cmd, flags, siteID)
			if err != nil {
				return err
			}
			next := make([]any, 0, len(posts)+1)
			for _, p := range posts {
				next = append(next, p)
			}
			next = append(next, post)
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{blogPostsField: next})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "blog-add", "status": status, "postId": post["id"], "response": parseJSONForOutput(resp)}, fmt.Sprintf("added blog post %q", title))
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "post title")
	cmd.Flags().StringVar(&body, "body", "", "post body (HTML)")
	cmd.Flags().StringVar(&author, "author", "", "author name")
	cmd.Flags().StringVar(&date, "date", "", "post date (default today, YYYY-MM-DD)")
	cmd.Flags().StringVar(&imageURL, "image-url", "", "hero image URL")
	cmd.Flags().StringVar(&imageKeywords, "image-keywords", "", "image keywords")
	return cmd
}

func newSitesBlogRmCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "rm <siteId> <postId>",
		Short:   "Remove a blog post.",
		Example: "  soloist-pp-cli sites blog rm <siteId> <postId>",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<siteId> <postId>")
			}
			siteID, postID := args[0], args[1]
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{"action": "blog-rm", "dry_run": true, "postId": postID}, "would remove blog post")
			}
			posts, err := loadBlogPosts(cmd, flags, siteID)
			if err != nil {
				return err
			}
			kept := make([]any, 0, len(posts))
			found := false
			for _, p := range posts {
				if stringValue(p["id"]) == postID {
					found = true
					continue
				}
				kept = append(kept, p)
			}
			if !found {
				return usageErr(fmt.Errorf("blog post %s not found", postID))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, status, err := patchWebsiteSettingsFields(cmd.Context(), c, siteID, map[string]any{blogPostsField: kept})
			if err != nil {
				return classifyAPIError(err, flags)
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{"action": "blog-rm", "status": status, "response": parseJSONForOutput(resp)}, "removed blog post")
		},
	}
}
