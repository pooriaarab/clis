// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored GA4 extensions. Not generated; safe across regen.
//
// Contents:
//   - client hook that mints a Google token before requests (ADC/SA/refresh)
//   - property alias registry (`alias` command) + ResolveProperty
//   - saved report specs (used by `report --save/--run`)
//   - shared date/kebab helpers
//   - the global --confirm mutation gate
//
// pp:data-source local — the alias/saved-report registry is a per-user local
// config file (aliases.json); `alias discover` refreshes it from the Admin API.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"google-analytics-solo-pp-cli/internal/client"
	"google-analytics-solo-pp-cli/internal/cliutil"
)

// mutationConfirm backs the global --confirm flag. Held as a package var so the
// gate can live entirely in hand-authored files (no edit to generated
// rootFlags). Registered as a persistent flag via the novel-command hook below.
var mutationConfirm bool

// --- client hook + command registration --------------------------------------

func init() {
	registerClientHook(func(c *client.Client) error {
		if c == nil || c.Config == nil {
			return nil
		}
		// Under the printing-press verifier, never touch real credentials.
		if cliutil.IsVerifyEnv() {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		// Non-fatal: a missing credential surfaces as a 401 with an actionable
		// message from the API. Commands that must have a token (alias
		// discover) call EnsureGoogleToken directly and surface the error.
		_ = c.Config.EnsureGoogleToken(ctx)
		return nil
	})

	// Register hand-authored commands + the mutation gate without editing the
	// generated root. Runs after all generated commands are added, so
	// applyConfirmGate sees the full endpoint tree.
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		root.PersistentFlags().BoolVar(&mutationConfirm, "confirm", false,
			"Actually perform mutating operations (Admin writes/deletes, mp send). Without it, mutations preview only.")
		root.AddCommand(newAliasCmd(flags))
		root.AddCommand(newMPCmd(flags))
		applyConfirmGate(root, flags)
	})
}

// --- alias registry ----------------------------------------------------------

type savedReport struct {
	Property string `json:"property,omitempty"`
	Dims     string `json:"dims,omitempty"`
	Metrics  string `json:"metrics,omitempty"`
	Since    string `json:"since,omitempty"`
	Until    string `json:"until,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type aliasStore struct {
	// Aliases maps a friendly name (e.g. "solo-prod") to a canonical
	// "properties/<numericId>" value.
	Aliases map[string]string `json:"aliases"`
	// Reports holds saved report specs keyed by name (report --save).
	Reports map[string]savedReport `json:"reports,omitempty"`
}

// aliasFilePath resolves the local, gitignored alias/report store. It never
// lives in the CLI repo: it is per-user config. GOOGLE_ANALYTICS_ALIASES
// overrides the path (used by tests and for the Work/.agents local config).
func aliasFilePath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("GOOGLE_ANALYTICS_ALIASES")); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "google-analytics-solo-pp-cli", "aliases.json"), nil
}

func loadAliases() (*aliasStore, error) {
	path, err := aliasFilePath()
	if err != nil {
		return nil, err
	}
	s := &aliasStore{Aliases: map[string]string{}, Reports: map[string]savedReport{}}
	// #nosec G304 -- path is this CLI's own per-user config file, from
	// aliasFilePath() (GOOGLE_ANALYTICS_ALIASES override or os.UserConfigDir()
	// + a fixed subpath), never untrusted request input.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if s.Aliases == nil {
		s.Aliases = map[string]string{}
	}
	if s.Reports == nil {
		s.Reports = map[string]savedReport{}
	}
	return s, nil
}

func saveAliases(s *aliasStore) error {
	path, err := aliasFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

var digitsRe = regexp.MustCompile(`^\d+$`)

// normalizeProperty turns "123", "properties/123", or "properties/123 " into
// the canonical "properties/123". Returns ok=false for non-numeric ids.
func normalizeProperty(v string) (string, bool) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "properties/")
	if !digitsRe.MatchString(v) {
		return "", false
	}
	return "properties/" + v, true
}

// ResolveProperty turns an alias or raw id into a canonical "properties/<id>".
func ResolveProperty(aliasOrID string) (string, error) {
	aliasOrID = strings.TrimSpace(aliasOrID)
	if aliasOrID == "" {
		return "", usageErr(fmt.Errorf("a --property alias or numeric id is required"))
	}
	if p, ok := normalizeProperty(aliasOrID); ok {
		return p, nil
	}
	store, err := loadAliases()
	if err != nil {
		return "", err
	}
	if p, ok := store.Aliases[aliasOrID]; ok {
		return p, nil
	}
	return "", usageErr(fmt.Errorf("unknown property alias %q: register it with 'google-analytics-solo-pp-cli alias add %s <numericId>' or run 'alias discover'", aliasOrID, aliasOrID))
}

var kebabRe = regexp.MustCompile(`[^a-z0-9]+`)

func kebab(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = kebabRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// parseSince converts "7d", "4w", "today", "yesterday", or a YYYY-MM-DD date
// into a GA4 date string. GA4 accepts NdaysAgo/today/yesterday and ISO dates.
func parseSince(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "7daysAgo", nil
	}
	switch s {
	case "today", "yesterday":
		return s, nil
	}
	if m := regexp.MustCompile(`^(\d+)d$`).FindStringSubmatch(s); m != nil {
		return m[1] + "daysAgo", nil
	}
	if m := regexp.MustCompile(`^(\d+)w$`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return strconv.Itoa(n*7) + "daysAgo", nil
	}
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(s) {
		return s, nil
	}
	return "", usageErr(fmt.Errorf("invalid --since %q: use Nd (e.g. 7d), Nw, today, yesterday, or YYYY-MM-DD", s))
}

func parseUntil(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "today", nil
	}
	if s == "today" || s == "yesterday" || regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(s) {
		return s, nil
	}
	if m := regexp.MustCompile(`^(\d+)d$`).FindStringSubmatch(s); m != nil {
		return m[1] + "daysAgo", nil
	}
	return "", usageErr(fmt.Errorf("invalid --until %q: use today, yesterday, Nd, or YYYY-MM-DD", s))
}

// sinceToDuration parses "Nd"/"Nw" (and plain "N" as days) into a lookback
// duration, used by trend to filter cached runs by age.
func sinceToDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	if m := regexp.MustCompile(`^(\d+)d?$`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return time.Duration(n) * 24 * time.Hour, nil
	}
	if m := regexp.MustCompile(`^(\d+)w$`).FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	}
	return 0, usageErr(fmt.Errorf("invalid --since %q: use Nd (e.g. 90d) or Nw", s))
}

// nameObjs turns "date,country" into [{"name":"date"},{"name":"country"}].
func nameObjs(csv string) []map[string]string {
	out := []map[string]string{}
	for _, part := range strings.Split(csv, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, map[string]string{"name": p})
		}
	}
	return out
}

// --- alias command -----------------------------------------------------------

func newAliasCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "alias",
		Short:       "Register GA4 property aliases (friendly name -> numeric id) in local config",
		Long:        "Register numeric GA4 property IDs under friendly names, stored in a local gitignored file, so every command can say --property solo-prod instead of properties/123456789.",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newAliasAddCmd(flags))
	cmd.AddCommand(newAliasListCmd(flags))
	cmd.AddCommand(newAliasRemoveCmd(flags))
	cmd.AddCommand(newAliasDiscoverCmd(flags))
	return cmd
}

func newAliasAddCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "add <name> <numericId>",
		Short:       "Register a property alias",
		Example:     "  google-analytics-solo-pp-cli alias add solo-prod 123456789",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 2 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("usage: alias add <name> <numericId>"))
			}
			prop, ok := normalizeProperty(args[1])
			if !ok {
				return usageErr(fmt.Errorf("%q is not a numeric property id (expected digits or properties/<digits>)", args[1]))
			}
			store, err := loadAliases()
			if err != nil {
				return err
			}
			store.Aliases[args[0]] = prop
			if err := saveAliases(store); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "registered %s -> %s\n", args[0], prop)
			return nil
		},
	}
}

func newAliasListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "list",
		Short:       "List registered property aliases",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			store, err := loadAliases()
			if err != nil {
				return err
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), store.Aliases, flags)
			}
			names := make([]string, 0, len(store.Aliases))
			for n := range store.Aliases {
				names = append(names, n)
			}
			sort.Strings(names)
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no aliases registered; add one with 'alias add <name> <id>' or 'alias discover'")
				return nil
			}
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", n, store.Aliases[n])
			}
			return nil
		},
	}
}

func newAliasRemoveCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "remove <name>",
		Short:       "Remove a property alias",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			if len(args) < 1 {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("usage: alias remove <name>"))
			}
			store, err := loadAliases()
			if err != nil {
				return err
			}
			delete(store.Aliases, args[0])
			if err := saveAliases(store); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", args[0])
			return nil
		},
	}
}

func newAliasDiscoverCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:         "discover",
		Short:       "Auto-register aliases from every GA4 property your account can see",
		Long:        "Calls the Admin API accountSummaries endpoint and registers an alias (kebab-cased display name -> numeric id) for every property the authenticated account can access.",
		Example:     "  google-analytics-solo-pp-cli alias discover",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would list accountSummaries and register property aliases")
				return nil
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			// accountSummaries is an Admin API endpoint on a different host than
			// the client's default (Data API) base_url, so use the absolute URL.
			raw, err := c.Get(ctx, "https://analyticsadmin.googleapis.com/v1alpha/accountSummaries", map[string]string{"pageSize": "200"})
			if err != nil {
				return fmt.Errorf("listing account summaries: %w", err)
			}
			var resp struct {
				AccountSummaries []struct {
					DisplayName       string `json:"displayName"`
					PropertySummaries []struct {
						Property    string `json:"property"`
						DisplayName string `json:"displayName"`
					} `json:"propertySummaries"`
				} `json:"accountSummaries"`
			}
			if err := json.Unmarshal(raw, &resp); err != nil {
				return fmt.Errorf("parsing account summaries: %w", err)
			}
			store, err := loadAliases()
			if err != nil {
				return err
			}
			added := map[string]string{}
			for _, acct := range resp.AccountSummaries {
				for _, ps := range acct.PropertySummaries {
					name := kebab(ps.DisplayName)
					if name == "" || ps.Property == "" {
						continue
					}
					store.Aliases[name] = ps.Property
					added[name] = ps.Property
				}
			}
			if err := saveAliases(store); err != nil {
				return err
			}
			if flags.asJSON || flags.agent {
				return printJSONFiltered(cmd.OutOrStdout(), added, flags)
			}
			if len(added) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no properties found for this account")
				return nil
			}
			names := make([]string, 0, len(added))
			for n := range added {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "registered %s -> %s\n", n, added[n])
			}
			return nil
		},
	}
}

// --- confirm gate ------------------------------------------------------------

// requireConfirm gates a mutating action behind --confirm. Without --confirm
// (or under dry-run / verify), it prints a preview and returns proceed=false so
// the command is a safe no-op. This protects live work GA4 config/data.
func requireConfirm(cmd *cobra.Command, flags *rootFlags, action string) (bool, error) {
	if dryRunOK(flags) || cliutil.IsVerifyEnv() {
		fmt.Fprintf(cmd.ErrOrStderr(), "would %s (dry-run; pass --confirm to execute)\n", action)
		return false, nil
	}
	if !mutationConfirm {
		fmt.Fprintf(cmd.ErrOrStderr(), "refusing to %s without --confirm (preview only; re-run with --confirm to execute)\n", action)
		return false, nil
	}
	return true, nil
}

// isMutationEndpoint decides whether a generated endpoint command mutates state.
// GA4 report endpoints are POST but read-only (runReport, batchRun*, etc.), so a
// bare "POST == mutation" rule would wrongly gate reports.
func isMutationEndpoint(method, path string) bool {
	m := strings.ToUpper(method)
	switch m {
	case "DELETE", "PATCH", "PUT":
		return true
	case "POST":
		readVerbs := []string{":runReport", ":runPivotReport", ":batchRunReports", ":batchRunPivotReports", ":runRealtimeReport", ":checkCompatibility", ":searchChangeHistoryEvents", ":runAccessReport"}
		for _, v := range readVerbs {
			if strings.Contains(path, v) {
				return false
			}
		}
		return true
	}
	return false
}

// applyConfirmGate wraps the RunE of every generated mutating endpoint command
// so it previews unless --confirm is passed. Called once from root.go after the
// command tree is built.
func applyConfirmGate(root *cobra.Command, flags *rootFlags) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		method := c.Annotations["pp:method"]
		path := c.Annotations["pp:path"]
		if method != "" && isMutationEndpoint(method, path) && c.RunE != nil {
			orig := c.RunE
			action := "execute " + c.CommandPath()
			c.RunE = func(cmd *cobra.Command, args []string) error {
				proceed, err := requireConfirm(cmd, flags, action)
				if err != nil {
					return err
				}
				if !proceed {
					return nil
				}
				return orig(cmd, args)
			}
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
}
