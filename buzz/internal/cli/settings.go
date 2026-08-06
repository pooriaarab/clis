package cli

import (
	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func settingsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "settings", Short: "Workspace settings commands"}

	get := &cobra.Command{
		Use:   "get",
		Short: "Get relay information",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			if resolved.RelayURL == "" {
				return inputError("relay URL is required")
			}
			raw, err := client.New(resolved.RelayURL, nil, nil).GetRelayInfo(cmd.Context())
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "get relay info failed", Err: err}
			}
			return opts.writeRawJSON(raw)
		},
	}

	set := &cobra.Command{
		Use:   "set",
		Short: "Set workspace profile settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			icon, _ := cmd.Flags().GetString("icon")
			clear, _ := cmd.Flags().GetBool("clear")
			iconChanged := cmd.Flags().Changed("icon")
			if iconChanged == clear {
				return inputError("exactly one of --icon or --clear is required")
			}
			if iconChanged && icon == "" {
				return inputError("--icon must be non-empty")
			}
			if clear {
				icon = ""
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindSetWorkspaceProfile, keys.PublicHex(), "", nostr.Tags{{"icon", icon}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign workspace settings event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	set.Flags().String("icon", "", "workspace icon URL")
	set.Flags().Bool("clear", false, "clear workspace icon")

	cmd.AddCommand(get, set)
	return cmd
}
