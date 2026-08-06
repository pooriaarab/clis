package cli

import (
	"strings"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func canvasCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "canvas", Short: "Canvas commands"}

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a channel canvas",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindCanvas}, "#h": []string{channel}, "limit": 1}})
		},
	}
	get.Flags().String("channel", "", "channel id")

	set := &cobra.Command{
		Use:   "set",
		Short: "Set a channel canvas",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			content, _ := cmd.Flags().GetString("content")
			if strings.TrimSpace(content) == "" {
				return inputError("--content is required")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindCanvas, keys.PublicHex(), content, nostr.Tags{{"h", channel}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign canvas event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	set.Flags().String("channel", "", "channel id")
	set.Flags().String("content", "", "canvas content")

	cmd.AddCommand(get, set)
	return cmd
}
