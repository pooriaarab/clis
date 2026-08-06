package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func reactionsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "reactions", Short: "Reaction commands"}

	add := &cobra.Command{
		Use:   "add",
		Short: "Add a reaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			target, err = validateHex64(target)
			if err != nil {
				return err
			}
			emoji, _ := cmd.Flags().GetString("emoji")
			if strings.TrimSpace(emoji) == "" {
				return inputError("--emoji is required")
			}
			emojiURL, _ := cmd.Flags().GetString("emoji-url")
			tags := nostr.Tags{{"e", target}}
			content := emoji
			if emojiURL != "" {
				shortcode, err := normalizeEmojiShortcode(emoji)
				if err != nil {
					return err
				}
				if err := validateEmojiURL(emojiURL); err != nil {
					return err
				}
				content = ":" + shortcode + ":"
				tags = append(tags, nostr.Tag{"emoji", shortcode, emojiURL})
			} else if utf8.RuneCountInString(emoji) > 64 {
				return inputError("emoji must be 64 characters or fewer")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindReaction, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign reaction event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	add.Flags().String("event", "", "event id")
	add.Flags().String("emoji", "", "emoji or shortcode")
	add.Flags().String("emoji-url", "", "custom emoji URL")

	remove := &cobra.Command{
		Use:   "remove",
		Short: "Remove a reaction",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			target, err = validateHex64(target)
			if err != nil {
				return err
			}
			emoji, _ := cmd.Flags().GetString("emoji")
			if strings.TrimSpace(emoji) == "" {
				return inputError("--emoji is required")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, keys, []client.Filter{{"kinds": []int{nostr.KindReaction}, "#e": []string{target}, "authors": []string{keys.PublicHex()}}})
			if err != nil {
				return err
			}
			var reactions []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(raw, &reactions); err != nil {
				return otherWrap("parse reaction query", err)
			}
			reactionID := ""
			for _, reaction := range reactions {
				if reaction.Content == emoji {
					reactionID = reaction.ID
					break
				}
			}
			if reactionID == "" {
				return ExitError{Code: ExitOther, Message: fmt.Sprintf("no reaction with emoji %q found for your pubkey on event %s", emoji, target)}
			}
			event := nostr.NewUnsignedEvent(nostr.KindDeletion, keys.PublicHex(), "", nostr.Tags{{"e", reactionID}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign reaction deletion event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	remove.Flags().String("event", "", "event id")
	remove.Flags().String("emoji", "", "emoji")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get reactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			target, err = validateHex64(target)
			if err != nil {
				return err
			}
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, nil, []client.Filter{{"kinds": []int{nostr.KindReaction}, "#e": []string{target}}})
			if err != nil {
				return err
			}
			var reactions []struct {
				Content string `json:"content"`
				PubKey  string `json:"pubkey"`
			}
			if err := json.Unmarshal(raw, &reactions); err != nil {
				return otherWrap("parse reactions", err)
			}
			type reactionGroup struct {
				Emoji   string   `json:"emoji"`
				Count   int      `json:"count"`
				PubKeys []string `json:"pubkeys"`
			}
			byEmoji := map[string]*reactionGroup{}
			for _, reaction := range reactions {
				emoji := reaction.Content
				if emoji == "" {
					emoji = "+"
				}
				group := byEmoji[emoji]
				if group == nil {
					group = &reactionGroup{Emoji: emoji}
					byEmoji[emoji] = group
				}
				group.Count++
				group.PubKeys = append(group.PubKeys, reaction.PubKey)
			}
			groups := make([]reactionGroup, 0, len(byEmoji))
			for _, group := range byEmoji {
				sort.Strings(group.PubKeys)
				groups = append(groups, *group)
			}
			sort.Slice(groups, func(i, j int) bool { return groups[i].Emoji < groups[j].Emoji })
			return opts.writeJSON(map[string]any{"reactions": groups})
		},
	}
	get.Flags().String("event", "", "event id")

	cmd.AddCommand(add, remove, get)
	return cmd
}

func validateHex64(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if len(value) != 64 {
		return "", inputError("value must be 64 hex characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", inputError("value must be 64 hex characters")
	}
	return value, nil
}

func normalizeEmojiShortcode(raw string) (string, error) {
	value := strings.ToLower(strings.Trim(strings.TrimSpace(raw), ":"))
	if value == "" {
		return "", inputError("emoji shortcode is required")
	}
	if len(value) > 64 {
		return "", inputError("emoji shortcode must be 64 characters or fewer")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return "", inputError("emoji shortcode may only contain ASCII letters, digits, '-' and '_'")
	}
	return value, nil
}

func validateEmojiURL(raw string) error {
	if raw == "" {
		return inputError("emoji-url is required")
	}
	if len(raw) > 2048 {
		return inputError("emoji-url must be 2048 bytes or fewer")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return inputError("emoji-url must start with http:// or https://")
	}
	return nil
}

func resolveNoKeys(opts *rootOptions) (config.Resolved, error) {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   opts.Identity,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return config.Resolved{}, otherWrap("resolve config", err)
	}
	return resolved, nil
}
