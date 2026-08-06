package cli

import (
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func patchesCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "patches", Short: "Send, get, list, and set status on git patches (NIP-34)"}

	send := &cobra.Command{
		Use:   "send",
		Short: "Send a git patch (NIP-34 kind:1617)",
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
			patchFile, err := requiredFlag(cmd, "patch-file")
			if err != nil {
				return err
			}
			content, err := readFileOrStdin(patchFile)
			if err != nil {
				return err
			}
			if strings.TrimSpace(content) == "" {
				return inputError("patch content must not be empty — refusing to publish an unappliable patch")
			}
			if err := validateContentSize(content, maxDiffBytes, "patch content"); err != nil {
				return err
			}

			euc, _ := cmd.Flags().GetString("euc")
			to, _ := cmd.Flags().GetStringArray("to")
			replyTo, _ := cmd.Flags().GetString("reply-to")
			root, _ := cmd.Flags().GetBool("root")
			rootRevision, _ := cmd.Flags().GetBool("root-revision")
			commit, _ := cmd.Flags().GetString("commit")
			parentCommit, _ := cmd.Flags().GetString("parent-commit")
			commitPGPSig, _ := cmd.Flags().GetString("commit-pgp-sig")
			committer, _ := cmd.Flags().GetString("committer")
			if root && rootRevision {
				return inputError("patch cannot be both --root and --root-revision")
			}

			tags := nostr.Tags{{"a", aTagValue(repoOwner, repoID)}}
			if euc != "" {
				if euc, err = validateCommitHex(euc, "euc"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"r", euc, "euc"})
			}
			tags = append(tags, nostr.Tag{"p", repoOwner})
			for _, recipient := range to {
				pk, err := validateHex64(recipient)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"p", pk})
			}
			if replyTo != "" {
				if replyTo, err = validateHex64(replyTo); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"e", replyTo, "", "reply"})
			}
			if root {
				tags = append(tags, nostr.Tag{"t", "root"})
			}
			if rootRevision {
				tags = append(tags, nostr.Tag{"t", "root-revision"})
			}
			if commit != "" {
				if commit, err = validateCommitHex(commit, "commit"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"commit", commit}, nostr.Tag{"r", commit})
			}
			if parentCommit != "" {
				if parentCommit, err = validateCommitHex(parentCommit, "parent_commit"); err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"parent-commit", parentCommit})
			}
			if commitPGPSig != "" {
				tags = append(tags, nostr.Tag{"commit-pgp-sig", commitPGPSig})
			}
			if committer != "" {
				name, email, ts, tz, err := parseCommitter(committer)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"committer", name, email, ts, tz})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindGitPatch, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign patch event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	send.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	send.Flags().String("repo-id", "", "repo identifier (d-tag)")
	send.Flags().String("patch-file", "", "path to a git format-patch file, or '-' to read from stdin")
	send.Flags().String("euc", "", "earliest-unique-commit of the repo")
	send.Flags().StringArray("to", nil, "additional recipient pubkey(s) (repeatable)")
	send.Flags().String("reply-to", "", "previous patch event id (series) or original root (revision)")
	send.Flags().Bool("root", false, "mark as the first patch of a new series")
	send.Flags().Bool("root-revision", false, "mark as the first patch of a new revision of an existing series")
	send.Flags().String("commit", "", "commit ID this patch produces when applied")
	send.Flags().String("parent-commit", "", "parent commit ID")
	send.Flags().String("commit-pgp-sig", "", "PGP signature of the commit")
	send.Flags().String("committer", "", "committer identity: 'name|email|timestamp|tz-offset-minutes'")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a patch by event id",
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			if event, err = validateHex64(event); err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindGitPatch}, "ids": []string{event}}})
		},
	}
	get.Flags().String("event", "", "patch event id (64-char hex)")

	list := &cobra.Command{
		Use:   "list",
		Short: "List patches for a repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, filter, err := repoScopedFilter(cmd, nostr.KindGitPatch)
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	list.Flags().String("repo-owner", "", "repo owner pubkey (64-char hex)")
	list.Flags().String("repo-id", "", "repo identifier (d-tag)")
	list.Flags().String("author", "", "filter by patch author pubkey")
	list.Flags().Int("limit", 0, "maximum number of results")

	status := &cobra.Command{
		Use:   "status",
		Short: "Set status on a patch (open/merged/closed/draft — NIP-34 kind:1630-1633)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requiredFlag(cmd, "root")
			if err != nil {
				return err
			}
			if root, err = validateHex64(root); err != nil {
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
			content, _ := cmd.Flags().GetString("content")
			body, err := readOrStdin(content)
			if err != nil {
				return err
			}
			if err := validateContentSize(body, maxContentBytes, "content"); err != nil {
				return err
			}

			tags, err := statusBaseTags(cmd, root)
			if err != nil {
				return err
			}
			if revision, _ := cmd.Flags().GetString("revision"); revision != "" {
				revision, err = validateHex64(revision)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"e", revision, "", "reply"})
			}

			q, _ := cmd.Flags().GetStringArray("q")
			mergeCommit, _ := cmd.Flags().GetString("merge-commit")
			appliedAsCommits, _ := cmd.Flags().GetStringArray("applied-as-commit")
			if kind != nostr.KindGitStatusMerged && (len(q) > 0 || mergeCommit != "" || len(appliedAsCommits) > 0) {
				return inputError("applied_patches/merge_commit/applied_as_commits only apply to the merged/resolved status")
			}
			for _, spec := range q {
				id, relay, pubkey, err := parseAppliedPatchRef(spec)
				if err != nil {
					return err
				}
				switch {
				case relay == "":
					tags = append(tags, nostr.Tag{"q", id})
				case pubkey == "":
					tags = append(tags, nostr.Tag{"q", id, relay})
				default:
					tags = append(tags, nostr.Tag{"q", id, relay, pubkey})
				}
			}
			if tags, err = appendMergeCommitTag(tags, mergeCommit); err != nil {
				return err
			}
			for i, commit := range appliedAsCommits {
				validated, err := validateCommitHex(commit, "applied_as_commit")
				if err != nil {
					return err
				}
				appliedAsCommits[i] = validated
			}
			if len(appliedAsCommits) > 0 {
				tags = append(tags, append(nostr.Tag{"applied-as-commits"}, appliedAsCommits...))
				for _, commit := range appliedAsCommits {
					tags = append(tags, nostr.Tag{"r", commit})
				}
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
	status.Flags().String("root", "", "root patch event id (first patch of the series/revision)")
	addStatusCommonFlags(status)
	status.Flags().String("content", "", "markdown context for the status change ('-' to read from stdin)")
	status.Flags().String("revision", "", "root id of the revision that was accepted (status=merged only)")
	status.Flags().StringArray("q", nil, "applied patch event id (status=merged only, repeatable)")
	status.Flags().String("merge-commit", "", "merge commit id (status=merged only)")
	status.Flags().StringArray("applied-as-commit", nil, "commit id applied to the target branch (status=merged only, repeatable)")

	cmd.AddCommand(send, get, list, status)
	return cmd
}

// repoScopedFilter builds the shared #a-scoped list filter used by
// patches/issues/pr list.
func repoScopedFilter(cmd *cobra.Command, kind int) (owner, id string, filter client.Filter, err error) {
	owner, err = requiredFlag(cmd, "repo-owner")
	if err != nil {
		return "", "", nil, err
	}
	if owner, err = validateHex64(owner); err != nil {
		return "", "", nil, err
	}
	id, err = requiredFlag(cmd, "repo-id")
	if err != nil {
		return "", "", nil, err
	}
	if id, err = validateRepoID(id); err != nil {
		return "", "", nil, err
	}
	filter = client.Filter{"kinds": []int{kind}, "#a": []string{aTagValue(owner, id)}}
	if author, _ := cmd.Flags().GetString("author"); author != "" {
		author, err = validateHex64(author)
		if err != nil {
			return "", "", nil, err
		}
		filter["authors"] = []string{author}
	}
	if label, _ := cmd.Flags().GetString("label"); label != "" {
		filter["#t"] = []string{label}
	}
	if cmd.Flags().Changed("limit") {
		limit, _ := cmd.Flags().GetInt("limit")
		filter["limit"] = limit
	}
	return owner, id, filter, nil
}
