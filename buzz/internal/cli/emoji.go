package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

const customEmojiDTag = "buzz:custom-emoji"

type emojiEntry struct {
	Shortcode string `json:"shortcode"`
	URL       string `json:"url"`
}

type emojiSetEvent struct {
	CreatedAt int64      `json:"created_at"`
	Tags      nostr.Tags `json:"tags"`
}

func emojiCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "emoji", Short: "Custom emoji commands"}

	list := &cobra.Command{
		Use:   "list",
		Short: "List workspace custom emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			entries, err := fetchWorkspaceEmoji(cmd.Context(), opts, resolved, nil)
			if err != nil {
				return err
			}
			return opts.writeJSON(map[string]any{"emojis": entries})
		},
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "Set a custom emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			shortcode, err := requiredFlag(cmd, "shortcode")
			if err != nil {
				return err
			}
			shortcode, err = normalizeEmojiShortcode(shortcode)
			if err != nil {
				return err
			}
			url, err := requiredFlag(cmd, "url")
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			entries, err := fetchOwnEmoji(cmd.Context(), opts, resolved, keys)
			if err != nil {
				return err
			}
			entries = removeEmojiEntry(entries, shortcode)
			entries = append(entries, emojiEntry{Shortcode: shortcode, URL: url})
			return publishOwnEmojiSet(cmd.Context(), opts, resolved, keys, entries)
		},
	}
	set.Flags().String("shortcode", "", "emoji shortcode")
	set.Flags().String("url", "", "emoji image URL")

	rm := &cobra.Command{
		Use:   "rm",
		Short: "Remove a custom emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			shortcode, err := requiredFlag(cmd, "shortcode")
			if err != nil {
				return err
			}
			shortcode, err = normalizeEmojiShortcode(shortcode)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			entries, err := fetchOwnEmoji(cmd.Context(), opts, resolved, keys)
			if err != nil {
				return err
			}
			next := removeEmojiEntry(entries, shortcode)
			if len(next) == len(entries) {
				return opts.writeJSON(map[string]any{"accepted": true, "message": "not present"})
			}
			return publishOwnEmojiSet(cmd.Context(), opts, resolved, keys, next)
		},
	}
	rm.Flags().String("shortcode", "", "emoji shortcode")

	export := &cobra.Command{
		Use:   "export",
		Short: "Export custom emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			scope, _ := cmd.Flags().GetString("scope")
			if scope != "own" && scope != "workspace" {
				return inputError("scope must be own or workspace")
			}
			resolved, keys, err := opts.resolveKeys(scope == "own")
			if err != nil {
				return err
			}
			var entries []emojiEntry
			if scope == "own" {
				entries, err = fetchOwnEmoji(cmd.Context(), opts, resolved, keys)
			} else {
				entries, err = fetchWorkspaceEmoji(cmd.Context(), opts, resolved, keys)
			}
			if err != nil {
				return err
			}
			sortEmojiEntries(entries)
			payload := map[string]any{"emojis": entries}
			if file == "" {
				return opts.writeJSON(payload)
			}
			b, err := json.Marshal(payload)
			if err != nil {
				return otherWrap("marshal emoji export", err)
			}
			if err := os.WriteFile(file, b, 0o600); err != nil {
				return otherWrap("write emoji export", err)
			}
			return nil
		},
	}
	export.Flags().String("file", "", "output file")
	export.Flags().String("scope", "own", "emoji scope: own or workspace")

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import custom emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			replace, _ := cmd.Flags().GetBool("replace")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			imported, err := readEmojiImport(file)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			finalSet := imported
			if !replace {
				existing, err := fetchOwnEmoji(cmd.Context(), opts, resolved, keys)
				if err != nil {
					return err
				}
				seen := map[string]struct{}{}
				for _, entry := range existing {
					seen[entry.Shortcode] = struct{}{}
				}
				finalSet = append([]emojiEntry{}, existing...)
				for _, entry := range imported {
					if _, ok := seen[entry.Shortcode]; ok {
						continue
					}
					finalSet = append(finalSet, entry)
					seen[entry.Shortcode] = struct{}{}
				}
			}
			if dryRun {
				if err := opts.writeJSON(map[string]any{"emojis": finalSet}); err != nil {
					return err
				}
				_, err := fmt.Fprintln(opts.stderr(), "(dry run — not published)")
				return err
			}
			return publishOwnEmojiSet(cmd.Context(), opts, resolved, keys, finalSet)
		},
	}
	importCmd.Flags().String("file", "", "input file")
	importCmd.Flags().Bool("replace", false, "replace existing emoji set")
	importCmd.Flags().Bool("dry-run", false, "print final set without publishing")

	cmd.AddCommand(list, set, rm, export, importCmd)
	return cmd
}

func fetchOwnEmoji(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair) ([]emojiEntry, error) {
	raw, err := opts.fetchQuery(ctx, resolved, keys, []client.Filter{{"kinds": []int{nostr.KindEmojiSet}, "#d": []string{customEmojiDTag}, "authors": []string{keys.PublicHex()}, "limit": 1}})
	if err != nil {
		return nil, err
	}
	var events []emojiSetEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, otherWrap("parse emoji set", err)
	}
	if len(events) == 0 {
		return []emojiEntry{}, nil
	}
	return emojiEntriesFromTags(events[0].Tags), nil
}

func publishOwnEmojiSet(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair, entries []emojiEntry) error {
	tags := nostr.Tags{{"d", customEmojiDTag}}
	for _, entry := range entries {
		tags = append(tags, nostr.Tag{"emoji", entry.Shortcode, entry.URL})
	}
	event := nostr.NewUnsignedEvent(nostr.KindEmojiSet, keys.PublicHex(), "", tags, 0)
	if err := event.Sign(keys); err != nil {
		return otherWrap("sign emoji set event", err)
	}
	return opts.publish(ctx, resolved, keys, event)
}

func fetchWorkspaceEmoji(ctx context.Context, opts *rootOptions, resolved config.Resolved, keys *nostr.KeyPair) ([]emojiEntry, error) {
	raw, err := opts.fetchQuery(ctx, resolved, keys, []client.Filter{{"kinds": []int{nostr.KindEmojiSet}, "#d": []string{customEmojiDTag}}})
	if err != nil {
		return nil, err
	}
	var events []emojiSetEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		return nil, otherWrap("parse workspace emoji sets", err)
	}
	return unionEmojiEntries(events), nil
}

func emojiEntriesFromTags(tags nostr.Tags) []emojiEntry {
	entries := []emojiEntry{}
	for _, tag := range tags {
		if len(tag) < 3 || tag[0] != "emoji" {
			continue
		}
		shortcode, err := normalizeEmojiShortcode(tag[1])
		if err != nil {
			continue
		}
		if tag[2] == "" {
			continue
		}
		entries = append(entries, emojiEntry{Shortcode: shortcode, URL: tag[2]})
	}
	return entries
}

func unionEmojiEntries(events []emojiSetEvent) []emojiEntry {
	type selected struct {
		entry     emojiEntry
		createdAt int64
	}
	byShortcode := map[string]selected{}
	for _, event := range events {
		for _, entry := range emojiEntriesFromTags(event.Tags) {
			current, ok := byShortcode[entry.Shortcode]
			if !ok || event.CreatedAt > current.createdAt || (event.CreatedAt == current.createdAt && entry.URL < current.entry.URL) {
				byShortcode[entry.Shortcode] = selected{entry: entry, createdAt: event.CreatedAt}
			}
		}
	}
	entries := make([]emojiEntry, 0, len(byShortcode))
	for _, selected := range byShortcode {
		entries = append(entries, selected.entry)
	}
	sortEmojiEntries(entries)
	return entries
}

func sortEmojiEntries(entries []emojiEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Shortcode == entries[j].Shortcode {
			return entries[i].URL < entries[j].URL
		}
		return entries[i].Shortcode < entries[j].Shortcode
	})
}

func removeEmojiEntry(entries []emojiEntry, shortcode string) []emojiEntry {
	out := entries[:0]
	for _, entry := range entries {
		if entry.Shortcode != shortcode {
			out = append(out, entry)
		}
	}
	return out
}

func readEmojiImport(file string) ([]emojiEntry, error) {
	var b []byte
	var err error
	if file != "" {
		b, err = os.ReadFile(file)
		if err != nil {
			return nil, inputWrap("read emoji import file", err)
		}
	} else {
		if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
			return nil, inputError("no input: provide --file or pipe JSON to stdin")
		}
		const maxImportBytes = 10 << 20
		b, err = io.ReadAll(io.LimitReader(os.Stdin, maxImportBytes+1))
		if err != nil {
			return nil, inputWrap("read emoji import stdin", err)
		}
		if len(b) > maxImportBytes {
			return nil, inputError("emoji import JSON must be 10MB or smaller")
		}
		if len(bytes.TrimSpace(b)) == 0 {
			return nil, inputError("no input: provide --file or pipe JSON to stdin")
		}
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, inputWrap("parse emoji import JSON", err)
	}
	rawEmojis, ok := top["emojis"]
	if !ok || len(bytes.TrimSpace(rawEmojis)) == 0 || bytes.TrimSpace(rawEmojis)[0] != '[' {
		return nil, inputError("missing top-level emojis array")
	}
	var rows []struct {
		Shortcode *string `json:"shortcode"`
		URL       *string `json:"url"`
	}
	if err := json.Unmarshal(rawEmojis, &rows); err != nil {
		return nil, inputWrap("parse emojis array", err)
	}
	seen := map[string]struct{}{}
	entries := make([]emojiEntry, 0, len(rows))
	for i, row := range rows {
		if row.Shortcode == nil {
			return nil, inputError(fmt.Sprintf("emojis[%d].shortcode is required", i))
		}
		if row.URL == nil {
			return nil, inputError(fmt.Sprintf("emojis[%d].url is required", i))
		}
		shortcode, err := normalizeEmojiShortcode(*row.Shortcode)
		if err != nil {
			return nil, inputError(fmt.Sprintf("emojis[%d].shortcode is invalid: %s", i, err.Error()))
		}
		url := strings.TrimSpace(*row.URL)
		if url == "" {
			return nil, inputError(fmt.Sprintf("emojis[%d].url is required", i))
		}
		if _, ok := seen[shortcode]; ok {
			continue
		}
		entries = append(entries, emojiEntry{Shortcode: shortcode, URL: url})
		seen[shortcode] = struct{}{}
	}
	return entries, nil
}
