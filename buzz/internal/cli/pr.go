package cli

import (
	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func prCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "pr", Short: "Open, update, list, and set status on git pull requests (NIP-34)"}

	open := &cobra.Command{
		Use:   "open",
		Short: "Open a git pull request (NIP-34 kind:1618)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoOwner, repoID, err := requiredRepoCoord(cmd)
			if err != nil {
				return err
			}
			subject, err := requiredFlag(cmd, "subject")
			if err != nil {
				return err
			}
			if len(subject) > 256 {
				return inputError("subject exceeds 256 characters")
			}
			body, _ := cmd.Flags().GetString("body")
			bodyFile, _ := cmd.Flags().GetString("body-file")
			content, err := readOptionalBody(body, bodyFile)
			if err != nil {
				return err
			}
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}
			commit, err := requiredFlag(cmd, "commit")
			if err != nil {
				return err
			}
			if commit, err = validateCommitHex(commit, "commit"); err != nil {
				return err
			}
			cloneURLs, _ := cmd.Flags().GetStringArray("clone")
			if len(cloneURLs) == 0 {
				return inputError("a pull request needs at least one --clone url where the tip commit can be fetched")
			}
			branchName, _ := cmd.Flags().GetString("branch-name")
			mergeBase, _ := cmd.Flags().GetString("merge-base")
			euc, _ := cmd.Flags().GetString("euc")
			labels, _ := cmd.Flags().GetStringArray("label")
			to, _ := cmd.Flags().GetStringArray("to")
			channel, _ := cmd.Flags().GetString("channel")
			revisionOf, _ := cmd.Flags().GetString("revision-of")

			tags := nostr.Tags{{"a", aTagValue(repoOwner, repoID)}}
			if euc != "" {
				if euc, err = validateCommitHex(euc, "euc"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"r", euc})
			}
			tags = append(tags, nostr.Tag{"p", repoOwner})
			for _, recipient := range to {
				pk, err := validateHex64(recipient)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"p", pk})
			}
			tags = append(tags, nostr.Tag{"subject", subject})
			for _, label := range labels {
				tags = append(tags, nostr.Tag{"t", label})
			}
			tags = append(tags, nostr.Tag{"c", commit})
			if channel != "" {
				parsed, err := uuid.Parse(channel)
				if err != nil {
					return inputError("channel_id must be a valid UUID: " + err.Error())
				}
				tags = append(tags, nostr.Tag{"h", parsed.String()})
			}
			tags = append(tags, append(nostr.Tag{"clone"}, cloneURLs...))
			if branchName != "" {
				tags = append(tags, nostr.Tag{"branch-name", branchName})
			}
			if mergeBase != "" {
				if mergeBase, err = validateCommitHex(mergeBase, "merge_base"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"merge-base", mergeBase})
			}
			if revisionOf != "" {
				if revisionOf, err = validateHex64(revisionOf); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"e", revisionOf})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitPullRequest, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign pull request event", err)
			}
			return opts.publishWithLink(cmd.Context(), resolved, keys, event, "link", pullRequestLink(event.ID, repoOwner, repoID))
		},
	}
	open.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	open.Flags().String("repo-id", "", "repo identifier (d-tag)")
	open.Flags().String("subject", "", "pull request subject/header")
	open.Flags().String("body", "", "pull request body markdown. Use '-' to read from stdin")
	open.Flags().String("body-file", "", "path to pull request body markdown, or '-' to read from stdin")
	open.Flags().String("commit", "", "tip commit of the PR branch")
	open.Flags().StringArray("clone", nil, "clone URL where the tip commit can be fetched (repeatable)")
	open.Flags().String("branch-name", "", "recommended branch name")
	open.Flags().String("merge-base", "", "most recent common ancestor with the target branch")
	open.Flags().String("euc", "", "earliest-unique-commit of the repo")
	open.Flags().StringArray("label", nil, "label (repeatable)")
	open.Flags().StringArray("to", nil, "additional recipient pubkey(s) (repeatable)")
	open.Flags().String("channel", "", "channel where this pull request originated (NIP-29 h-tag)")
	open.Flags().String("revision-of", "", "root patch event id this PR revises")

	update := &cobra.Command{
		Use:   "update",
		Short: "Update a git pull request tip (NIP-34 kind:1619)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoOwner, repoID, err := requiredRepoCoord(cmd)
			if err != nil {
				return err
			}
			pr, err := requiredFlag(cmd, "pr")
			if err != nil {
				return err
			}
			if pr, err = validateHex64(pr); err != nil {
				return err
			}
			prAuthor, err := requiredFlag(cmd, "pr-author")
			if err != nil {
				return err
			}
			if prAuthor, err = validateHex64(prAuthor); err != nil {
				return err
			}
			commit, err := requiredFlag(cmd, "commit")
			if err != nil {
				return err
			}
			if commit, err = validateCommitHex(commit, "commit"); err != nil {
				return err
			}
			cloneURLs, _ := cmd.Flags().GetStringArray("clone")
			if len(cloneURLs) == 0 {
				return inputError("a pull request update needs at least one --clone url where the tip commit can be fetched")
			}
			body, _ := cmd.Flags().GetString("body")
			bodyFile, _ := cmd.Flags().GetString("body-file")
			content, err := readOptionalBody(body, bodyFile)
			if err != nil {
				return err
			}
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}
			mergeBase, _ := cmd.Flags().GetString("merge-base")
			euc, _ := cmd.Flags().GetString("euc")
			to, _ := cmd.Flags().GetStringArray("to")

			tags := nostr.Tags{{"a", aTagValue(repoOwner, repoID)}}
			if euc != "" {
				if euc, err = validateCommitHex(euc, "euc"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"r", euc})
			}
			tags = append(tags, nostr.Tag{"p", repoOwner})
			for _, recipient := range to {
				pk, err := validateHex64(recipient)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"p", pk})
			}
			tags = append(tags, nostr.Tag{"E", pr}, nostr.Tag{"P", prAuthor}, nostr.Tag{"c", commit})
			tags = append(tags, append(nostr.Tag{"clone"}, cloneURLs...))
			if mergeBase != "" {
				if mergeBase, err = validateCommitHex(mergeBase, "merge_base"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"merge-base", mergeBase})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitPrUpdate, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign pull request update event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	update.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	update.Flags().String("repo-id", "", "repo identifier (d-tag)")
	update.Flags().String("pr", "", "pull request event id being updated")
	update.Flags().String("pr-author", "", "pull request author's pubkey")
	update.Flags().String("commit", "", "updated tip commit of the PR branch")
	update.Flags().StringArray("clone", nil, "clone URL where the updated tip commit can be fetched (repeatable)")
	update.Flags().String("body", "", "markdown context for the update. Use '-' to read from stdin")
	update.Flags().String("body-file", "", "path to markdown context for the update, or '-' to read from stdin")
	update.Flags().String("merge-base", "", "most recent common ancestor with the target branch")
	update.Flags().String("euc", "", "earliest-unique-commit of the repo")
	update.Flags().StringArray("to", nil, "additional recipient pubkey(s) (repeatable)")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a PR by event id",
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if event, err = validateHex64(event); err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindGitPullRequest}, "ids": []string{event}}})
		},
	}
	get.Flags().String("event", "", "PR event id (64-char hex)")

	list := &cobra.Command{
		Use:   "list",
		Short: "List PRs for a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, filter, err := repoScopedFilter(cmd, nostr.KindGitPullRequest)
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	list.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	list.Flags().String("repo-id", "", "repo identifier (d-tag)")
	list.Flags().String("author", "", "filter by PR author pubkey")
	list.Flags().String("label", "", "filter by label")
	list.Flags().Int("limit", 0, "maximum number of results")

	status := &cobra.Command{
		Use:   "status",
		Short: "Set status on a PR (open/merged/closed/draft — NIP-34 kind:1630-1633)",
		RunE: func(cmd *cobra.Command, args []string) error {
			pr, err := requiredFlag(cmd, "pr")
			if err != nil {
				return err
			}
			if pr, err = validateHex64(pr); err != nil {
				return err
			}
			statusWord, err := requiredFlag(cmd, "status")
			if err != nil {
				return err
			}
			kind, err := gitStatusKindRestricted(statusWord, patchOrPrStatusWords)
			if err != nil {
				return err
			}
			body, _ := cmd.Flags().GetString("body")
			bodyFile, _ := cmd.Flags().GetString("body-file")
			content, err := readOptionalBody(body, bodyFile)
			if err != nil {
				return err
			}
			if err := validateContentSize(content, maxContentBytes, "content"); err != nil {
				return err
			}
			tags, err := statusBaseTags(cmd, pr)
			if err != nil {
				return err
			}
			mergeCommit, _ := cmd.Flags().GetString("merge-commit")
			if tags, err = appendMergeCommitTag(tags, mergeCommit); err != nil {
				return err
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign status event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	status.Flags().String("pr", "", "pull request event id")
	addStatusCommonFlags(status)
	status.Flags().String("body", "", "markdown context for the status change. Use '-' to read from stdin")
	status.Flags().String("body-file", "", "path to markdown context for the status change, or '-' to read from stdin")
	status.Flags().String("merge-commit", "", "merge commit id (status=merged only)")

	cmd.AddCommand(open, update, get, list, status)
	return cmd
}

// requiredRepoCoord reads and validates the shared --repo-owner/--repo-id
// pair for pr open/update.
func requiredRepoCoord(cmd *cobra.Command) (owner, id string, err error) {
	owner, err = requiredFlag(cmd, "repo-owner")
	if err != nil {
		return "", "", err
	}
	if owner, err = validateHex64(owner); err != nil {
		return "", "", err
	}
	id, err = requiredFlag(cmd, "repo-id")
	if err != nil {
		return "", "", err
	}
	if id, err = validateRepoID(id); err != nil {
		return "", "", err
	}
	return owner, id, nil
}
