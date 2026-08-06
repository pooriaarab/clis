package cli

import (
	"encoding/json"
	"time"

	"buzz-cli/internal/config"
	"github.com/spf13/cobra"
)

func inviteCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "invite", Short: "Invite commands"}

	create := &cobra.Command{
		Use:   "create",
		Short: "Create an invite",
		RunE: func(cmd *cobra.Command, args []string) error {
			ttlSecs, _ := cmd.Flags().GetUint64("ttl-secs")
			maxUses, _ := cmd.Flags().GetInt("max-uses")
			if cmd.Flags().Changed("max-uses") && maxUses < 0 {
				return inputError("max-uses must be greater than or equal to zero")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			body := map[string]any{}
			if cmd.Flags().Changed("ttl-secs") {
				body["ttl_secs"] = ttlSecs
			}
			if cmd.Flags().Changed("max-uses") {
				body["max_uses"] = maxUses
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.Post(cmd.Context(), "/api/invites", body)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "create invite failed", Err: err}
			}
			var response struct {
				Code          string `json:"code"`
				ExpiresAt     int64  `json:"expires_at"`
				MaxUses       *int   `json:"max_uses"`
				UsesRemaining *int   `json:"uses_remaining"`
				URL           string `json:"url"`
			}
			if err := json.Unmarshal(raw, &response); err != nil {
				return ExitError{Code: ExitRelay, Message: "parse invite response failed", Err: err}
			}
			record := config.InviteRecord{
				Code:      response.Code,
				URL:       response.URL,
				ExpiresAt: response.ExpiresAt,
				CreatedAt: time.Now().Unix(),
			}
			if response.MaxUses != nil {
				record.MaxUses = *response.MaxUses
			}
			if response.UsesRemaining != nil {
				record.UsesRemaining = *response.UsesRemaining
			}
			if err := resolved.File.AppendInvite(resolved.ConfigPath, record); err != nil {
				return otherWrap("save invite", err)
			}
			return opts.writeRawJSON(raw)
		},
	}
	create.Flags().Uint64("ttl-secs", 0, "invite TTL in seconds")
	create.Flags().Int("max-uses", 0, "maximum invite uses")

	claim := &cobra.Command{
		Use:   "claim",
		Short: "Claim an invite",
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := requiredFlag(cmd, "code")
			if err != nil {
				return err
			}
			policyReceipt, _ := cmd.Flags().GetString("policy-receipt")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			body := map[string]any{"code": code}
			if policyReceipt != "" {
				body["policy_receipt"] = policyReceipt
			}
			relayClient, err := restClientFromResolved(resolved, keys)
			if err != nil {
				return err
			}
			raw, err := relayClient.Post(cmd.Context(), "/api/invites/claim", body)
			if err != nil {
				return ExitError{Code: ExitRelay, Message: "claim invite failed", Err: err}
			}
			return opts.writeRawJSON(raw)
		},
	}
	claim.Flags().String("code", "", "invite code")
	claim.Flags().String("policy-receipt", "", "policy receipt")

	list := &cobra.Command{
		Use:   "list",
		Short: "List locally-created invites",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveNoKeys(opts)
			if err != nil {
				return err
			}
			invites := resolved.File.Invites
			if invites == nil {
				invites = []config.InviteRecord{}
			}
			return opts.writeJSON(invites)
		},
	}

	cmd.AddCommand(create, claim, list)
	return cmd
}
