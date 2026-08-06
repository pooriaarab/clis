package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"buzz-cli/internal/client"
	"github.com/spf13/cobra"
)

var validFeedTypes = map[string]struct{}{
	"mentions":       {},
	"needs_action":   {},
	"activity":       {},
	"agent_activity": {},
}

func feedCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "feed", Short: "Feed commands"}

	get := &cobra.Command{
		Use:   "get",
		Short: "Get your feed",
		RunE: func(cmd *cobra.Command, args []string) error {
			since, _ := cmd.Flags().GetInt64("since")
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				return inputError("limit must be greater than zero")
			}
			if limit > 50 {
				limit = 50
			}
			typesRaw, _ := cmd.Flags().GetString("types")
			feedTypes, err := parseFeedTypes(typesRaw)
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			filter := client.Filter{"#p": []string{keys.PublicHex()}, "limit": limit}
			if cmd.Flags().Changed("since") {
				filter["since"] = since
			}
			if len(feedTypes) > 0 {
				filter["feed_types"] = feedTypes
			}
			raw, err := opts.fetchQuery(cmd.Context(), resolved, keys, []client.Filter{filter})
			if err != nil {
				return err
			}
			var events []map[string]any
			if err := json.Unmarshal(raw, &events); err != nil {
				return otherWrap("parse feed events", err)
			}
			sort.SliceStable(events, func(i, j int) bool {
				return eventCreatedAt(events[i]) > eventCreatedAt(events[j])
			})
			if opts.Format == "compact" {
				compact := make([]map[string]any, 0, len(events))
				for _, event := range events {
					compact = append(compact, map[string]any{
						"id":         event["id"],
						"content":    event["content"],
						"created_at": event["created_at"],
					})
				}
				return opts.writeJSON(compact)
			}
			return opts.writeJSON(events)
		},
	}
	get.Flags().Int64("since", 0, "minimum created_at timestamp")
	get.Flags().Int("limit", 20, "max results")
	get.Flags().String("types", "", "feed types CSV")
	cmd.AddCommand(get)
	return cmd
}

func parseFeedTypes(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if _, ok := validFeedTypes[value]; !ok {
			return nil, inputError(fmt.Sprintf("invalid feed type %q; valid types are mentions, needs_action, activity, agent_activity", value))
		}
		out = append(out, value)
	}
	return out, nil
}

func eventCreatedAt(event map[string]any) float64 {
	switch value := event["created_at"].(type) {
	case float64:
		return value
	case int64:
		return float64(value)
	case int:
		return float64(value)
	case json.Number:
		n, _ := value.Float64()
		return n
	default:
		return 0
	}
}
