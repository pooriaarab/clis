// Package cli wires the cobra command tree.
package cli

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/submit/internal/output"
)

type cobraCmd = cobra.Command

// Version is the CLI version.
const Version = "0.1.0"

// Env holds what every command needs. Tests and main inject the pieces.
type Env struct {
	P      *output.Printer
	Getenv func(string) string
	DryRun bool
}

// Main runs the CLI and returns the process exit code.
func Main(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	env := &Env{P: &output.Printer{Out: stdout, Err: stderr}, Getenv: getenv}
	root := newRoot(env)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	err := root.Execute()
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) && ee.Printed {
		return ee.Code
	}
	if env.P.JSON || hasFlag(args, "--json") {
		_ = (&output.Printer{Out: stdout}).Object(map[string]any{"ok": false, "error": err.Error()})
	} else {
		env.P.Warnf("error: %s", err)
	}
	return exitCode(err)
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
		if v, ok := strings.CutPrefix(a, name+"="); ok {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
			return true
		}
	}
	return false
}

func newRoot(env *Env) *cobra.Command {
	root := &cobra.Command{
		Use:           "submit",
		Short:         "Directory-submission copilot and tracker: copy kit, status store, listing checks",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().BoolVar(&env.P.JSON, "json", false, "print one JSON object instead of text")
	root.PersistentFlags().BoolVar(&env.DryRun, "dry-run", false, "show what would change without changing it (no writes, no form posts)")
	root.AddCommand(
		versionCmd(env),
	)
	return root
}

func versionCmd(env *Env) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Args:  cobra.NoArgs,
		RunE: run(func([]string) error {
			if env.P.JSON {
				return env.P.Object(map[string]any{"version": Version})
			}
			env.P.Linef("submit %s", Version)
			return nil
		}),
	}
}
