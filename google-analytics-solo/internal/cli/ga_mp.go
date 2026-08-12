// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored Measurement Protocol commands. Not generated; safe across regen.
//
// The Measurement Protocol is a DIFFERENT API from Data/Admin: it ingests
// client-side events and authenticates with an api_secret + measurement_id, not
// OAuth. `mp send` writes to live analytics, so it is confirm-gated. `mp debug`
// validates a payload without recording it.

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const mpBaseURL = "https://www.google-analytics.com"

func mpResolveAuth(cmd *cobra.Command, measurementID string) (string, string, error) {
	if measurementID == "" {
		measurementID = os.Getenv("GA4_MEASUREMENT_ID")
	}
	secret := os.Getenv("GA4_MP_API_SECRET")
	if measurementID == "" {
		return "", "", usageErr(fmt.Errorf("--measurement-id is required (or set GA4_MEASUREMENT_ID)"))
	}
	if secret == "" {
		return "", "", usageErr(fmt.Errorf("GA4_MP_API_SECRET is required (create one in GA4 Admin > Data Streams > Measurement Protocol API secrets)"))
	}
	return measurementID, secret, nil
}

// mpBuildPayload assembles the Measurement Protocol event envelope.
func mpBuildPayload(clientID, userID, event, paramsJSON string) (map[string]any, error) {
	params := map[string]any{}
	if strings.TrimSpace(paramsJSON) != "" {
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return nil, usageErr(fmt.Errorf("--params must be a JSON object: %w", err))
		}
	}
	payload := map[string]any{
		"client_id": clientID,
		"events":    []map[string]any{{"name": event, "params": params}},
	}
	if userID != "" {
		payload["user_id"] = userID
	}
	return payload, nil
}

func mpPost(ctx context.Context, endpoint, measurementID, secret string, payload map[string]any) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	q := url.Values{}
	q.Set("measurement_id", measurementID)
	q.Set("api_secret", secret)
	u := endpoint + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, nil
}

func newMPCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "mp",
		Short:       "Measurement Protocol: send or validate client-side GA4 events",
		Long:        "Send events into a GA4 property via the Measurement Protocol. Auth is an api_secret (GA4_MP_API_SECRET) plus a measurement id, not OAuth. 'mp send' writes live data and requires --confirm; 'mp debug' only validates.",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newMPDebugCmd(flags))
	cmd.AddCommand(newMPSendCmd(flags))
	return cmd
}

func mpFlags(cmd *cobra.Command, mid, cid, uid, event, params *string) {
	cmd.Flags().StringVar(mid, "measurement-id", "", "GA4 measurement id (e.g. G-XXXXXXXX) or set GA4_MEASUREMENT_ID")
	cmd.Flags().StringVar(cid, "client-id", "", "Client id for the event (required)")
	cmd.Flags().StringVar(uid, "user-id", "", "Optional user id")
	cmd.Flags().StringVar(event, "event", "", "Event name (required, e.g. sign_up)")
	cmd.Flags().StringVar(params, "params", "", "Event params as a JSON object")
}

func newMPDebugCmd(flags *rootFlags) *cobra.Command {
	var mid, cid, uid, event, params string
	cmd := &cobra.Command{
		Use:         "debug",
		Short:       "Validate a Measurement Protocol event without recording it",
		Example:     "  google-analytics-solo-pp-cli mp debug --measurement-id G-XXXX --client-id 123.456 --event sign_up",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would validate an MP event via /debug/mp/collect")
				return nil
			}
			if cid == "" || event == "" {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--client-id and --event are required"))
			}
			measurementID, secret, err := mpResolveAuth(cmd, mid)
			if err != nil {
				return err
			}
			payload, err := mpBuildPayload(cid, uid, event, params)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			status, body, err := mpPost(ctx, mpBaseURL+"/debug/mp/collect", measurementID, secret, payload)
			if err != nil {
				return fmt.Errorf("validating event: %w", err)
			}
			var v any
			if json.Unmarshal(body, &v) == nil {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"status": status, "validation": v}, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "status=%d\n%s\n", status, string(body))
			return nil
		},
	}
	mpFlags(cmd, &mid, &cid, &uid, &event, &params)
	return cmd
}

func newMPSendCmd(flags *rootFlags) *cobra.Command {
	var mid, cid, uid, event, params string
	cmd := &cobra.Command{
		Use:         "send",
		Short:       "Send a client-side event into GA4 (writes live data; requires --confirm)",
		Example:     "  google-analytics-solo-pp-cli mp send --measurement-id G-XXXX --client-id 123.456 --event sign_up --confirm",
		Annotations: map[string]string{"mcp:read-only": "false", "pp:method": "POST", "pp:path": "/mp/collect"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if cid == "" || event == "" {
				// Surface the requirement even before the confirm gate so the
				// preview is meaningful.
				if !dryRunOK(flags) {
					_ = cmd.Usage()
					return usageErr(fmt.Errorf("--client-id and --event are required"))
				}
			}
			proceed, err := requireConfirm(cmd, flags, "send an event to GA4")
			if err != nil {
				return err
			}
			if !proceed {
				return nil
			}
			measurementID, secret, err := mpResolveAuth(cmd, mid)
			if err != nil {
				return err
			}
			payload, err := mpBuildPayload(cid, uid, event, params)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			status, body, err := mpPost(ctx, mpBaseURL+"/mp/collect", measurementID, secret, payload)
			if err != nil {
				return fmt.Errorf("sending event: %w", err)
			}
			// MP /mp/collect returns 204 with an empty body on success.
			out := map[string]any{"status": status, "sent": status >= 200 && status < 300}
			if len(body) > 0 {
				out["body"] = string(body)
			}
			return printJSONFiltered(cmd.OutOrStdout(), out, flags)
		},
	}
	mpFlags(cmd, &mid, &cid, &uid, &event, &params)
	return cmd
}
