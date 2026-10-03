package cli

import (
	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/kit"
)

func kitCmd(env *Env) *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "kit --sites sites.yaml --out kit/",
		Short: "Render the per-site copy kit as markdown + JSON",
		Args:  cobra.NoArgs,
	}
	load := sitesFlag(env, cmd)
	cmd.Flags().StringVar(&out, "out", "", "output directory for <id>.md + <id>.json (required)")
	_ = cmd.MarkFlagRequired("out")
	cmd.RunE = run(func([]string) error {
		list, path, err := load()
		if err != nil {
			return err
		}
		if env.DryRun {
			paths := kit.Planned(list, out)
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "dry_run": true, "sites": path, "paths": paths})
			}
			for _, p := range paths {
				env.P.Linef("would write %s", p)
			}
			env.P.Linef("dry run: nothing was written")
			return nil
		}
		paths, err := kit.Render(list, out)
		if err != nil {
			return err
		}
		if env.P.JSON {
			return env.P.Object(map[string]any{"ok": true, "sites": path, "paths": paths})
		}
		for _, p := range paths {
			env.P.Linef("wrote %s", p)
		}
		return nil
	})
	return cmd
}
