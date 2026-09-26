// Package cli wires the walmart-ca command tree.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/spf13/cobra"

	"walmart-ca-cli/internal/walmart"
)

// Execute runs the root command.
func Execute() error {
	root := &cobra.Command{
		Use:           "walmart-ca",
		Short:         "Read walmart.ca order history via your own browser session",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	auth := &cobra.Command{Use: "auth", Short: "Session management"}
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a session from a browser cURL capture",
		Long:  "In signed-in Chrome: DevTools > Network, refresh " + walmart.Base + "/en/orders, filter for PurchaseHistoryV2, right-click > Copy > Copy as cURL. Then: walmart-ca auth import --file curl.txt (or pipe on stdin). Stored in ~/.walmart-ca/session.json (0600); values are never printed.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			file, _ := cmd.Flags().GetString("file")
			raw, err := os.ReadFile(file)
			if file == "" {
				raw, err = io.ReadAll(cmd.InOrStdin())
			}
			if err != nil {
				return err
			}
			sess, err := walmart.ParseCurl(string(raw))
			if err != nil {
				return err
			}
			if err := walmart.SaveSession(sess); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "imported %d cookies, query hash %.12s…\n", len(sess.Cookies), sess.Hash)
			return nil
		},
	}
	importCmd.Flags().String("file", "", "cURL capture file (default: stdin)")
	auth.AddCommand(importCmd)

	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Verify session state and walmart.ca reachability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			sess, serr := walmart.LoadSession()
			st := "!! missing: run `walmart-ca auth import`"
			if serr == nil {
				st = fmt.Sprintf("ok: %d cookies, saved %s", len(sess.Cookies), sess.SavedAt)
			}
			fmt.Fprintln(out, "session:", st)
			_, derr := net.DefaultResolver.LookupHost(cmd.Context(), "www.walmart.ca")
			dns := "ok"
			if derr != nil {
				dns = "!! " + derr.Error()
			}
			fmt.Fprintln(out, "dns: www.walmart.ca", dns)
			return nil
		},
	}

	ordersList := &cobra.Command{
		Use:   "list",
		Short: "List past orders, newest first",
		RunE: func(cmd *cobra.Command, _ []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			search, _ := cmd.Flags().GetString("search")
			asJSON, _ := cmd.Flags().GetBool("json")
			sess, err := walmart.LoadSession()
			if err != nil {
				return err
			}
			c := walmart.NewClient(sess)
			var groups []walmart.OrderGroup
			for cursor := ""; len(groups) < limit; {
				n := limit - len(groups)
				if n > 20 {
					n = 20
				}
				g, next, err := c.History(cmd.Context(), n, cursor, search)
				if err != nil {
					return err
				}
				groups = append(groups, g...)
				if cursor = next; cursor == "" {
					break
				}
			}
			out := cmd.OutOrStdout()
			if asJSON {
				data, _ := json.MarshalIndent(groups, "", "  ")
				fmt.Fprintln(out, string(data))
				return nil
			}
			for _, g := range groups {
				fmt.Fprintln(out, g.String())
			}
			fmt.Fprintf(out, "(%d orders)\n", len(groups))
			return nil
		},
	}
	ordersList.Flags().Int("limit", 20, "max orders to show")
	ordersList.Flags().String("search", "", "filter orders by item text")
	ordersList.Flags().Bool("json", false, "machine-readable output")
	orders := &cobra.Command{Use: "orders", Short: "Order history"}
	orders.AddCommand(ordersList)

	root.AddCommand(auth, doctor, orders)
	return root.Execute()
}
