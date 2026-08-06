package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"buzz-cli/internal/types"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// channelSummary mirrors the bundled CLI's ChannelSummary projection
// (buzz-cli/src/commands/channels.rs, `channels search`) so agents get the
// same stable, scriptable shape rather than a raw kind:39000 event.
type channelSummary struct {
	ChannelID   string  `json:"channel_id"`
	Name        string  `json:"name"`
	ChannelType *string `json:"channel_type"`
	Visibility  *string `json:"visibility"`
	Archived    bool    `json:"archived"`
	About       *string `json:"about"`
	Topic       *string `json:"topic"`
	Purpose     *string `json:"purpose"`
}

// channelSummaryFromTags parses a kind:39000 event's tags into a
// channelSummary. Returns ok=false when the required `d` (channel id) or
// `name` tags are missing, matching ChannelSummary::from_event.
//
// Note: the oracle reads channel type from a bare "t" tag, not
// "channel_type" (the tag `channels create` actually writes) — copied
// verbatim rather than "fixed" here, since this command must match the
// bundled CLI's observed behavior exactly.
func channelSummaryFromTags(tags nostr.Tags) (channelSummary, bool) {
	var cs channelSummary
	var hasID, hasName bool
	for _, tag := range tags {
		if len(tag) == 0 {
			continue
		}
		key := tag[0]
		val := ""
		hasVal := len(tag) >= 2
		if hasVal {
			val = tag[1]
		}
		switch key {
		case "d":
			if hasVal {
				cs.ChannelID = val
				hasID = true
			}
		case "name":
			if hasVal {
				cs.Name = val
				hasName = true
			}
		case "t":
			if hasVal {
				v := val
				cs.ChannelType = &v
			}
		case "private":
			v := "private"
			cs.Visibility = &v
		case "public":
			v := "public"
			cs.Visibility = &v
		case "about":
			if hasVal {
				v := val
				cs.About = &v
			}
		case "topic":
			if hasVal {
				v := val
				cs.Topic = &v
			}
		case "purpose":
			if hasVal {
				v := val
				cs.Purpose = &v
			}
		case "archived":
			cs.Archived = hasVal && val == "true"
		}
	}
	if !hasID || !hasName {
		return channelSummary{}, false
	}
	return cs, true
}

func channelsSearchCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search channels by human-readable name",
		RunE: func(cmd *cobra.Command, args []string) error {
			query, err := requiredFlag(cmd, "query")
			if err != nil {
				return err
			}
			exact, _ := cmd.Flags().GetBool("exact")
			includeArchived, _ := cmd.Flags().GetBool("include-archived")
			limit, _ := cmd.Flags().GetInt("limit")

			events, err := opts.queryEvents(cmd.Context(), []client.Filter{{
				"kinds": []int{nostr.KindChannelMetadata},
				"limit": positiveOr(limit, 1000),
			}})
			if err != nil {
				return err
			}

			needle := strings.ToLower(query)
			matches := make([]channelSummary, 0, len(events))
			for _, e := range events {
				cs, ok := channelSummaryFromTags(e.Tags)
				if !ok {
					continue
				}
				if !includeArchived && cs.Archived {
					continue
				}
				hay := strings.ToLower(cs.Name)
				if exact {
					if hay != needle {
						continue
					}
				} else if !strings.Contains(hay, needle) {
					continue
				}
				matches = append(matches, cs)
			}
			sort.Slice(matches, func(i, j int) bool {
				if matches[i].Name != matches[j].Name {
					return matches[i].Name < matches[j].Name
				}
				return matches[i].ChannelID < matches[j].ChannelID
			})
			return opts.writeJSON(matches)
		},
	}
	cmd.Flags().String("query", "", "Search query (case-insensitive substring of channel name)")
	cmd.Flags().Bool("exact", false, "Require an exact case-insensitive match instead of substring")
	cmd.Flags().Bool("include-archived", false, "Include archived channels in results")
	cmd.Flags().Int("limit", 1000, "Maximum number of channel-metadata events to fetch from the relay")
	return cmd
}

func channelsUpdateCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update channel name, description, or ephemeral TTL",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("invalid channel UUID", err)
			}
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			ttl, _ := cmd.Flags().GetInt64("ttl")
			noTTL, _ := cmd.Flags().GetBool("no-ttl")
			nameSet := cmd.Flags().Changed("name")
			descSet := cmd.Flags().Changed("description")
			ttlSet := cmd.Flags().Changed("ttl")
			if ttlSet && noTTL {
				return inputError("--ttl conflicts with --no-ttl")
			}
			if !nameSet && !descSet && !ttlSet && !noTTL {
				return inputError("at least one field required (--name, --description, --ttl, --no-ttl)")
			}

			tags := nostr.Tags{{"h", channel}}
			if nameSet {
				canonical := types.CanonicalChannelName(name)
				if strings.TrimSpace(canonical) == "" {
					return inputError("channel name is required")
				}
				tags = append(tags, nostr.Tag{"name", canonical})
			}
			if descSet {
				tags = append(tags, nostr.Tag{"about", description})
			}
			switch {
			case ttlSet:
				if ttl <= 0 {
					return inputError("--ttl must be a positive number of seconds")
				}
				tags = append(tags, nostr.Tag{"ttl", strconv.FormatInt(ttl, 10)})
			case noTTL:
				tags = append(tags, nostr.Tag{"ttl", ""})
			}

			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindNIP29EditMetadata, keys.PublicHex(), "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign update channel event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	cmd.Flags().String("name", "", "New channel name")
	cmd.Flags().String("description", "", "New channel description")
	cmd.Flags().Int64("ttl", 0, "Make the channel ephemeral: seconds until the relay archives it after the last message. Conflicts with --no-ttl")
	cmd.Flags().Bool("no-ttl", false, "Clear an existing TTL, making the channel permanent")
	return cmd
}

// publishChannelEditTag builds and publishes a kind:9002 (NIP-29 edit
// metadata) event carrying a single ["h", channel] + one extra tag —
// shared by topic/purpose/archive/unarchive.
func (opts *rootOptions) publishChannelEvent(ctx context.Context, channel string, kind int, extra nostr.Tags) error {
	if _, err := uuid.Parse(channel); err != nil {
		return inputWrap("invalid channel UUID", err)
	}
	resolved, keys, err := opts.resolveKeys(true)
	if err != nil {
		return err
	}
	tags := append(nostr.Tags{{"h", channel}}, extra...)
	event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), "", tags, 0)
	if err := event.Sign(keys); err != nil {
		return otherWrap("sign channel event", err)
	}
	return opts.publish(ctx, resolved, keys, event)
}

func channelsTopicCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Set the channel topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			topic, err := requiredFlag(cmd, "topic")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29EditMetadata, nostr.Tags{{"topic", topic}})
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	cmd.Flags().String("topic", "", "New topic text")
	return cmd
}

func channelsPurposeCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purpose",
		Short: "Set the channel purpose",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			purpose, err := requiredFlag(cmd, "purpose")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29EditMetadata, nostr.Tags{{"purpose", purpose}})
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	cmd.Flags().String("purpose", "", "New purpose text")
	return cmd
}

func channelsLeaveCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Leave a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29LeaveRequest, nil)
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	return cmd
}

func channelsArchiveCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29EditMetadata, nostr.Tags{{"archived", "true"}})
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	return cmd
}

func channelsUnarchiveCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unarchive",
		Short: "Unarchive a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29EditMetadata, nostr.Tags{{"archived", "false"}})
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	return cmd
}

func channelsDeleteCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a channel permanently",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.publishChannelEvent(cmd.Context(), channel, nostr.KindNIP29DeleteGroup, nil)
		},
	}
	cmd.Flags().String("channel", "", "Channel UUID")
	return cmd
}

func channelsSetAddPolicyCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-add-policy",
		Short: "Set your channel addition policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			policy, err := requiredFlag(cmd, "policy")
			if err != nil {
				return err
			}
			switch policy {
			case "anyone", "owner_only", "nobody":
			default:
				return inputError(fmt.Sprintf("--policy must be 'anyone', 'owner_only', or 'nobody' (got: %s)", policy))
			}
			// Deployment gate, matching buzz-cli's cmd_set_add_policy: an env-configured
			// allowlist restricts which policies this CLI path may submit. Bypassable by
			// submitting the kind:10100 event directly — full enforcement is relay-side
			// and out of scope here, same as the bundled CLI.
			if allowedRaw := strings.TrimSpace(os.Getenv("BUZZ_ACP_ALLOWED_CHANNEL_ADD_POLICIES")); allowedRaw != "" {
				allowed := compactStrings([]string{allowedRaw})
				if len(allowed) > 0 && !containsString(allowed, policy) {
					return inputError(fmt.Sprintf(
						"channel_add_policy %q is not permitted on this deployment (BUZZ_ACP_ALLOWED_CHANNEL_ADD_POLICIES=%s)",
						policy, allowedRaw))
				}
			}
			content, err := json.Marshal(map[string]string{"channel_add_policy": policy})
			if err != nil {
				return otherWrap("encode policy content", err)
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindAgentProfile, keys.PublicHex(), string(content), nostr.Tags{}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign policy event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	cmd.Flags().String("policy", "", "Policy: anyone | owner_only | nobody")
	return cmd
}
