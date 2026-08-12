package cli

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
)

var rawBases = map[string]string{
	"admin":       "https://analyticsadmin.googleapis.com/v1beta",
	"admin-alpha": "https://analyticsadmin.googleapis.com/v1alpha",
	"data":        "https://analyticsdata.googleapis.com/v1beta",
	"data-alpha":  "https://analyticsdata.googleapis.com/v1alpha",
}

// rawURL joins a host alias with a path, or passes a full URL through untouched.
func rawURL(host, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	base, ok := rawBases[host]
	if !ok {
		base = rawBases["admin"]
	}
	return base + "/" + strings.TrimPrefix(path, "/")
}

func rawIsMutating(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "OPTIONS":
		return false
	default:
		return true
	}
}

// rawNeedsBody reports methods where GA4 custom-method endpoints (e.g. :archive)
// 404 on a zero-byte body; such requests get a default "{}" body.
func rawNeedsBody(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PATCH", "PUT":
		return true
	default:
		return false
	}
}

// decodeJSONOr returns s parsed as JSON so it nests as a value, or the raw
// string when it is not valid JSON (e.g. empty body).
func decodeJSONOr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}

func newRawCmd(flags *rootFlags) *cobra.Command {
	var host, body string
	var confirm bool
	c := &cobra.Command{
		Use:   "raw <METHOD> <PATH>",
		Short: "Call ANY GA4 Admin/Data endpoint (100% API escape hatch). Mutations need --confirm.",
		Long: `Call any GA4 REST endpoint directly. PATH is joined with --host, or pass a full URL.

Reads (GET/HEAD) run immediately. Mutations (POST/PATCH/PUT/DELETE) are a dry-run
unless --confirm is set, so an accidental write is a no-op that prints the request.

Examples:
  raw GET  properties/379350696/customDimensions
  raw POST properties/379350696/customDimensions --body '{"parameterName":"plan","displayName":"Plan","scope":"USER"}' --confirm
  raw DELETE properties/379350696/customDimensions/1234 --confirm`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			target := rawURL(host, args[1])
			if rawNeedsBody(method) && strings.TrimSpace(body) == "" {
				body = "{}"
			}
			if rawIsMutating(method) && !confirm {
				return output(cmd, flags, map[string]any{
					"dry_run": true, "method": method, "url": target,
					"body": decodeJSONOr(body),
					"note": "re-run with --confirm to execute this mutating request",
				}, "")
			}
			cl, _, err := flags.newClient()
			if err != nil {
				return err
			}
			resp, st, err := cl.Raw(context.Background(), method, target, []byte(body))
			if err != nil {
				return err
			}
			return output(cmd, flags, map[string]any{"status": st, "url": target, "response": decodeJSONOr(string(resp))}, "")
		},
	}
	c.Flags().StringVar(&host, "host", "admin", "Base host: admin, admin-alpha, data, data-alpha (ignored if PATH is a full URL)")
	c.Flags().StringVar(&body, "body", "", "Request body JSON (POST/PATCH/PUT)")
	c.Flags().BoolVar(&confirm, "confirm", false, "Execute a mutating request instead of dry-running it")
	return c
}
