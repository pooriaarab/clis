package cli

import (
	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func issuesCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "issues", Short: "Create, get, list, and set status on git issues (NIP-34)"}

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a git issue (NIP-34 kind:1621)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoOwner, err := requiredFlag(cmd, "repo-owner")
			if err != nil {
				return err
			}
			if repoOwner, err = validateHex64(repoOwner); err != nil {
				return err
			}
			repoID, err := requiredFlag(cmd, "repo-id")
			if err != nil {
				return err
			}
			if repoID, err = validateRepoID(repoID); err != nil {
				return err
			}
			title, err := requiredFlag(cmd, "title")
			if err != nil {
				return err
			}
			if len(title) > 256 {
				return inputError("subject exceeds 256 characters")
			}
			contentFlag, err := requiredFlag(cmd, "content")
			if err != nil {
				return err
			}
			content, err := readOrStdin(contentFlag)
			if err != nil {
				return err
			}
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}
			labels, _ := cmd.Flags().GetStringArray("label")
			to, _ := cmd.Flags().GetStringArray("to")

			tags := nostr.Tags{{"a", aTagValue(repoOwner, repoID)}, {"p", repoOwner}}
			for _, recipient := range to {
				pk, err := validateHex64(recipient)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"p", pk})
			}
			tags = append(tags, nostr.Tag{"subject", title})
			for _, label := range labels {
				tags = append(tags, nostr.Tag{"t", label})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitIssue, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign issue event", err)
			}
			return opts.publishWithLink(cmd.Context(), resolved, keys, event, "link", issueLink(event.ID, repoOwner, repoID))
		},
	}
	create.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	create.Flags().String("repo-id", "", "repo identifier (d-tag)")
	create.Flags().String("title", "", "issue title")
	create.Flags().String("content", "", "issue body, markdown. Use '-' to read from stdin")
	create.Flags().StringArray("label", nil, "label (repeatable)")
	create.Flags().StringArray("to", nil, "additional recipient pubkey(s) (repeatable)")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get an issue by event id",
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if event, err = validateHex64(event); err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindGitIssue}, "ids": []string{event}}})
		},
	}
	get.Flags().String("event", "", "issue event id (64-char hex)")

	list := &cobra.Command{
		Use:   "list",
		Short: "List issues for a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, filter, err := repoScopedFilter(cmd, nostr.KindGitIssue)
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	list.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	list.Flags().String("repo-id", "", "repo identifier (d-tag)")
	list.Flags().String("author", "", "filter by issue author pubkey")
	list.Flags().String("label", "", "filter by label")
	list.Flags().Int("limit", 0, "maximum number of results")

	status := &cobra.Command{
		Use:   "status",
		Short: "Set status on an issue (open/resolved/closed/draft — NIP-34 kind:1630-1633)",
		RunE: func(cmd *cobra.Command, args []string) error {
			issue, err := requiredFlag(cmd, "issue")
			if err != nil {
				return err
			}
			if issue, err = validateHex64(issue); err != nil {
				return err
			}
			statusWord, err := requiredFlag(cmd, "status")
			if err != nil {
				return err
			}
			kind, err := gitStatusKindRestricted(statusWord, issueStatusWords)
			if err != nil {
				return err
			}
			content, _ := cmd.Flags().GetString("content")
			body, err := readOrStdin(content)
			if err != nil {
				return err
			}
			if err := validateContentSize(body, maxContentBytes, "content"); err != nil {
				return err
			}
			tags, err := statusBaseTags(cmd, issue)
			if err != nil {
				return err
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), body, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign status event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	status.Flags().String("issue", "", "issue event id")
	addStatusCommonFlags(status)
	status.Flags().String("content", "", "markdown context for the status change ('-' to read from stdin)")

	cmd.AddCommand(create, get, list, status)
	return cmd
}
