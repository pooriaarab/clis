// Copyright 2026 Pooria Arab and contributors. Licensed under Apache-2.0. See LICENSE.
// Hand-authored novel command. generate --force preserves implemented bodies.
//
// `properties add <name> <id>` registers a property alias. It shares the local
// alias store with the `alias` command family; this entry point exists because
// the README/SKILL advertise `properties add solo-prod 123456789`.

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newNovelPropertiesAddCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "add <name> <numericId>",
		Short:       "Register a GA4 property alias (friendly name -> numeric id) in local config.",
		Example:     "  google-analytics-solo-pp-cli properties add solo-prod 123456789",
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
				return usageErr(fmt.Errorf("usage: properties add <name> <numericId>"))
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
	return cmd
}
