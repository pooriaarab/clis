package cli

// buzz moderation — community moderation queue, enforcement, and audit.
//
// Mutations (ban/unban/timeout/untimeout/resolve) are signed command events
// (kinds 9040-9044, buzz-core/src/kind.rs) submitted via POST /events; the
// relay validates, authorizes (owner/admin only), and executes them
// directly — they are never stored. Reads (reports/restricted/audit) hit
// dedicated mod-only, NIP-98-authed relay endpoints under /moderation/*
// (buzz-relay/src/api/bridge.rs, buzz-relay/src/router.rs). Tags mirror
// buzz-sdk/src/builders.rs build_moderation_*.

import (
	"fmt"
	"strconv"
	"time"

	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func moderationCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "moderation", Short: "Community moderation commands"}

	reports := &cobra.Command{
		Use:   "reports",
		Short: "List reports in the moderation queue (newest first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetInt64("limit")
			path := fmt.Sprintf("/moderation/reports?limit=%d", limit)
			if status != "" {
				path += "&status=" + status
			}
			return opts.moderationGet(cmd, path)
		},
	}
	reports.Flags().String("status", "", "filter by status: open | resolved | dismissed | escalated (default: all)")
	reports.Flags().Int64("limit", 50, "maximum number of reports to return")

	resolve := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve or dismiss a report (kind 9044)",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := requiredFlag(cmd, "report")
			if err != nil {
				return err
			}
			report, err = validateHex64(report)
			if err != nil {
				return err
			}
			status, err := requiredFlag(cmd, "status")
			if err != nil {
				return err
			}
			if status != "resolved" && status != "dismissed" {
				return inputError("status must be resolved or dismissed (got: " + status + ")")
			}
			action, err := requiredFlag(cmd, "action")
			if err != nil {
				return err
			}
			switch action {
			case "delete", "kick", "ban", "timeout", "dismiss", "escalate":
			default:
				return inputError("action must be delete, kick, ban, timeout, dismiss, or escalate (got: " + action + ")")
			}
			reason, _ := cmd.Flags().GetString("reason")
			tags := nostr.Tags{{"report", report}, {"status", status}, {"action", action}}
			if reason != "" {
				tags = append(tags, nostr.Tag{"reason", reason})
			}
			return opts.moderationSubmit(cmd, nostr.KindModerationResolve, tags)
		},
	}
	resolve.Flags().String("report", "", "hex event id of the kind:1984 report being resolved")
	resolve.Flags().String("status", "", "resolution status: resolved | dismissed")
	resolve.Flags().String("action", "", "action taken: delete | kick | ban | timeout | dismiss | escalate")
	resolve.Flags().String("reason", "", "optional reason — relayed to the reporter, so keep it tombstone-safe")

	ban := &cobra.Command{
		Use:   "ban",
		Short: "Ban a member from the community (kind 9040)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := moderationTargetTags(cmd)
			if err != nil {
				return err
			}
			expiry, err := resolveExpiry(cmd)
			if err != nil {
				return err
			}
			if expiry != nil {
				tags = append(tags, nostr.Tag{"expiration", strconv.FormatUint(*expiry, 10)})
			}
			if reason, _ := cmd.Flags().GetString("reason"); reason != "" {
				tags = append(tags, nostr.Tag{"reason", reason})
			}
			return opts.moderationSubmit(cmd, nostr.KindModerationBan, tags)
		},
	}
	ban.Flags().String("pubkey", "", "target member pubkey (hex)")
	ban.Flags().Uint64("expires-in", 0, "ban duration in seconds from now (omit for a permanent ban)")
	ban.Flags().Uint64("expires-at", 0, "absolute ban expiry as a unix timestamp (seconds)")
	ban.Flags().String("reason", "", "optional private ban reason (audit only)")

	unban := &cobra.Command{
		Use:   "unban",
		Short: "Lift a member's ban (kind 9041)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := moderationTargetTags(cmd)
			if err != nil {
				return err
			}
			return opts.moderationSubmit(cmd, nostr.KindModerationUnban, tags)
		},
	}
	unban.Flags().String("pubkey", "", "target member pubkey (hex)")

	timeout := &cobra.Command{
		Use:   "timeout",
		Short: "Time out a member — a write-block, not a disconnect (kind 9042)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := moderationTargetTags(cmd)
			if err != nil {
				return err
			}
			expiry, err := resolveExpiry(cmd)
			if err != nil {
				return err
			}
			if expiry == nil {
				return inputError("timeout requires --expires-in or --expires-at")
			}
			tags = append(tags, nostr.Tag{"expiration", strconv.FormatUint(*expiry, 10)})
			if reason, _ := cmd.Flags().GetString("reason"); reason != "" {
				tags = append(tags, nostr.Tag{"reason", reason})
			}
			return opts.moderationSubmit(cmd, nostr.KindModerationTimeout, tags)
		},
	}
	timeout.Flags().String("pubkey", "", "target member pubkey (hex)")
	timeout.Flags().Uint64("expires-in", 0, "timeout duration in seconds from now")
	timeout.Flags().Uint64("expires-at", 0, "absolute timeout expiry as a unix timestamp (seconds)")
	timeout.Flags().String("reason", "", "optional private timeout reason (audit only)")

	untimeout := &cobra.Command{
		Use:   "untimeout",
		Short: "Clear a member's timeout early (kind 9043)",
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := moderationTargetTags(cmd)
			if err != nil {
				return err
			}
			return opts.moderationSubmit(cmd, nostr.KindModerationUntimeout, tags)
		},
	}
	untimeout.Flags().String("pubkey", "", "target member pubkey (hex)")

	restricted := &cobra.Command{
		Use:   "restricted",
		Short: "List currently-restricted members (active ban or timeout)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.moderationGet(cmd, "/moderation/restricted")
		},
	}

	audit := &cobra.Command{
		Use:   "audit",
		Short: "Read the moderation audit trail (newest first)",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt64("limit")
			return opts.moderationGet(cmd, fmt.Sprintf("/moderation/audit?limit=%d", limit))
		},
	}
	audit.Flags().Int64("limit", 50, "maximum number of audit rows to return")

	cmd.AddCommand(reports, resolve, ban, unban, timeout, untimeout, restricted, audit)
	return cmd
}

// moderationTargetTags validates --pubkey and returns the base `p` tag list
// shared by ban/unban/timeout/untimeout.
func moderationTargetTags(cmd *cobra.Command) (nostr.Tags, error) {
	pubkey, err := requiredFlag(cmd, "pubkey")
	if err != nil {
		return nil, err
	}
	pubkey, err = validateHex64(pubkey)
	if err != nil {
		return nil, err
	}
	return nostr.Tags{{"p", pubkey}}, nil
}

// resolveExpiry mirrors resolve_expiry in moderation.rs: --expires-in wins,
// otherwise --expires-at, otherwise None. Cobra's mutual --expires-at
// exclusion with --expires-in is enforced here since cobra flag groups
// don't support cross-command conflicts as tersely as clap.
func resolveExpiry(cmd *cobra.Command) (*uint64, error) {
	expiresIn, _ := cmd.Flags().GetUint64("expires-in")
	expiresAt, _ := cmd.Flags().GetUint64("expires-at")
	hasIn := cmd.Flags().Changed("expires-in")
	hasAt := cmd.Flags().Changed("expires-at")
	if hasIn && hasAt {
		return nil, inputError("--expires-in and --expires-at are mutually exclusive")
	}
	if hasIn {
		v := uint64(time.Now().Unix()) + expiresIn
		return &v, nil
	}
	if hasAt {
		return &expiresAt, nil
	}
	return nil, nil
}

// moderationSubmit signs and submits a moderation command event, printing
// the normalized write response (mirrors normalize_write_response).
func (opts *rootOptions) moderationSubmit(cmd *cobra.Command, kind int, tags nostr.Tags) error {
	resolved, keys, err := opts.resolveKeys(true)
	if err != nil {
		return err
	}
	event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), "", tags, 0)
	if err := event.Sign(keys); err != nil {
		return otherWrap("sign moderation event", err)
	}
	return opts.submitAndNormalize(cmd.Context(), resolved, keys, event)
}

// moderationGet performs a NIP-98-signed GET against a mod-only relay
// endpoint and prints the raw JSON body unchanged.
func (opts *rootOptions) moderationGet(cmd *cobra.Command, path string) error {
	resolved, keys, err := opts.resolveKeys(true)
	if err != nil {
		return err
	}
	relayClient, err := restClientFromResolved(resolved, keys)
	if err != nil {
		return err
	}
	raw, err := relayClient.Get(cmd.Context(), path)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "moderation read failed", Err: err}
	}
	return opts.writeRawJSON(raw)
}
