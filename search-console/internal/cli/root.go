// Package cli wires the cobra command tree.
package cli

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/httpx"
	"github.com/pooriaarab/clis/search-console/internal/output"
)

type cobraCmd = cobra.Command

// Version is the CLI version.
const Version = "0.1.0"

// Env holds what every command needs. Tests and main inject the pieces.
type Env struct {
	P      *output.Printer
	Getenv func(string) string
	Client *httpx.Client
	DryRun bool
}

// Emit prints a result. In JSON mode it prints v as one object. In dry-run mode
// it adds the calls that were not sent, in both modes.
func (e *Env) Emit(v any, human func()) error {
	if e.P.JSON {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		m := map[string]any{}
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		if e.DryRun {
			m["dry_run"] = true
			m["calls"] = e.Client.Calls
		}
		return e.P.Object(m)
	}
	human()
	if e.DryRun {
		for _, c := range e.Client.Calls {
			e.P.Linef("dry-run: %s %s", c.Method, c.URL)
			if c.Body != "" {
				e.P.Linef("         %s", c.Body)
			}
		}
	}
	return nil
}

// Main runs the CLI and returns the process exit code.
func Main(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	env := &Env{P: &output.Printer{Out: stdout, Err: stderr}, Getenv: getenv, Client: httpx.New(false)}
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
		if a == name || strings.HasPrefix(a, name+"=") {
			return true
		}
	}
	return false
}

func newRoot(env *Env) *cobra.Command {
	root := &cobra.Command{
		Use:           "search-console",
		Short:         "Launch a domain into Google Search Console, Bing Webmaster Tools and IndexNow",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().BoolVar(&env.P.JSON, "json", false, "print one JSON object instead of text")
	root.PersistentFlags().BoolVar(&env.DryRun, "dry-run", false, "print the HTTP calls without sending them")
	root.PersistentPreRun = func(*cobra.Command, []string) { env.Client = httpx.New(env.DryRun) }
	root.AddCommand(versionCmd(env), sitemapCmd(env), authCmd(env))
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
			env.P.Linef("search-console %s", Version)
			return nil
		}),
	}
}
