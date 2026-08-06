package cli

import (
	"encoding/json"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func dmsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "dms", Short: "Direct message commands"}

	list := &cobra.Command{
		Use:   "list",
		Short: "List direct messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				return inputError("limit must be greater than zero")
			}
			if limit > 200 {
				limit = 200
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, keys, []client.Filter{{"kinds": []int{nostr.KindDMCreated}, "#p": []string{keys.PublicHex()}, "limit": limit}})
			if err != nil {
				return err
			}
			var events []struct {
				CreatedAt int64      `json:"created_at"`
				Tags      nostr.Tags `json:"tags"`
			}
			if err := json.Unmarshal(raw, &events); err != nil {
				return otherWrap("parse dm list", err)
			}
			out := make([]map[string]any, 0, len(events))
			for _, event := range events {
				out = append(out, map[string]any{
					"dm_id":        firstTagValue(event.Tags, "d"),
					"participants": tagValues(event.Tags, "p"),
					"created_at":   event.CreatedAt,
				})
			}
			return opts.writeJSON(out)
		},
	}
	list.Flags().Int("limit", 50, "max results")

	open := &cobra.Command{
		Use:   "open",
		Short: "Open a direct message",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkeys, _ := cmd.Flags().GetStringSlice("pubkey")
			pubkeys = compactStrings(pubkeys)
			if len(pubkeys) == 0 || len(pubkeys) > 8 {
				return inputError("pubkey must be provided 1 to 8 times")
			}
			for i, pubkey := range pubkeys {
				normalized, err := validateHex64(pubkey)
				if err != nil {
					return inputError("pubkey must be 64 hex characters")
				}
				pubkeys[i] = normalized
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			dmID := uuid.NewString()
			tags := nostr.Tags{}
			for _, pubkey := range pubkeys {
				tags = append(tags, nostr.Tag{"p", pubkey})
			}
			tags = append(tags, nostr.Tag{"d", dmID})
			event := nostr.NewUnsignedEvent(nostr.KindDMOpen, keys.PublicHex(), "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign dm open event", err)
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.PostEvent(cmd.Context(), event)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "publish dm open event failed", Err: err}
			}
			resolvedID := dmID
			if relayID := relayAssignedChannelID(raw); relayID != "" {
				resolvedID = relayID
			}
			out := map[string]any{}
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &out); err != nil {
					return otherWrap("parse dm open response", err)
				}
			}
			out["dm_id"] = resolvedID
			if _, ok := out["accepted"]; !ok {
				out["accepted"] = true
			}
			return opts.writeJSON(out)
		},
	}
	open.Flags().StringSlice("pubkey", nil, "participant pubkey")

	addMember := &cobra.Command{
		Use:   "add-member",
		Short: "Add a direct message member",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("parse channel uuid", err)
			}
			pubkey, err := requiredFlag(cmd, "pubkey")
			if err != nil {
				return err
			}
			pubkey, err = validateHex64(pubkey)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindDMAddMember, keys.PublicHex(), "", nostr.Tags{{"h", channel}, {"p", pubkey}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign dm add-member event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	addMember.Flags().String("channel", "", "dm channel id")
	addMember.Flags().String("pubkey", "", "member pubkey")

	hide := &cobra.Command{
		Use:   "hide",
		Short: "Hide a direct message",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			if _, err := uuid.Parse(channel); err != nil {
				return inputWrap("parse channel uuid", err)
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindDMHide, keys.PublicHex(), "", nostr.Tags{{"h", channel}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign dm hide event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	hide.Flags().String("channel", "", "dm channel id")

	cmd.AddCommand(list, open, addMember, hide)
	return cmd
}

func firstTagValue(tags nostr.Tags, key string) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == key {
			return tag[1]
		}
	}
	return ""
}

func tagValues(tags nostr.Tags, key string) []string {
	out := []string{}
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == key {
			out = append(out, tag[1])
		}
	}
	return out
}

func relayAssignedChannelID(raw json.RawMessage) string {
	var response map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &response) != nil {
		return ""
	}
	message, _ := response["message"].(string)
	if !strings.HasPrefix(message, "response:") {
		return ""
	}
	var nested map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(message, "response:")), &nested); err != nil {
		return ""
	}
	channelID, _ := nested["channel_id"].(string)
	return channelID
}
