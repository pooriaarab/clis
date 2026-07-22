package cli

import "github.com/spf13/cobra"

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		root.AddCommand(newSitesCmd(flags))
	})
}

func newSitesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "sites",
		Short:       "Edit draft website content.",
		Annotations: map[string]string{"pp:typed-exit-codes": "0,2,3,4,5"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSitesSectionsCmd(flags))
	cmd.AddCommand(newSitesPublishCmd(flags))
	cmd.AddCommand(newSitesThemeCmd(flags))
	cmd.AddCommand(newSitesSettingsCmd(flags))
	cmd.AddCommand(newSitesPagesCmd(flags))
	cmd.AddCommand(newSitesBlogCmd(flags))
	return cmd
}
