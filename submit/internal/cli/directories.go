package cli

import (
	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/directories"
)

func directoriesCmd(env *Env) *cobra.Command {
	var wave string
	cmd := &cobra.Command{
		Use:   "directories",
		Short: "Show the built-in ranked directory list and runbook waves",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List the 32 ranked directories, optionally one wave",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			data, err := directories.Load()
			if err != nil {
				return err
			}
			var entries []directories.Entry
			for _, e := range data.Directories {
				if wave != "" && e.Wave != wave {
					continue
				}
				entries = append(entries, e)
			}
			if wave != "" && len(entries) == 0 {
				return usagef("unknown wave %q (want 1-5, geo or company)", wave)
			}
			if env.P.JSON {
				return env.P.Object(map[string]any{"ok": true, "waves": data.Waves, "directories": entries})
			}
			table := [][]string{{"RANK", "ID", "NAME", "COST", "WAVE"}}
			for _, e := range entries {
				table = append(table, []string{itoa(e.Rank), e.ID, e.Name, e.Cost, e.Wave})
			}
			env.P.Table(table)
			return nil
		}),
	}
	list.Flags().StringVar(&wave, "wave", "", "only wave 1-5, geo or company")
	cmd.AddCommand(list)
	return cmd
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
