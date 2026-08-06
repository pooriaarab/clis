// Package mcp exposes buzz-cli's command tree as MCP tools over stdio.
//
// It does not reimplement any protocol/event logic. Every tool call runs
// buzz-cli's own cobra command tree in-process (see runCLI below) — the
// exact same code path `buzz <args...>` runs — so config/identity/relay
// resolution, event building/signing, and JSON output are all identical to
// the CLI's, by construction rather than by duplication.
package mcp

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"buzz-cli/internal/cli"
)

// runCLI builds a fresh buzz-cli root command, runs it with argv, and
// returns its stdout. Errors surface the same human-readable message the
// CLI would print to stderr (cli.ExitError.Message), without duplicating
// buzz-cli's own error-JSON formatting — the MCP result envelope carries
// error text separately (mcplib.NewToolResultError), so a second layer of
// JSON wrapping here would just be noise for the calling agent.
func runCLI(ctx context.Context, argv []string) (string, error) {
	root, opts := cli.NewRootCommand()
	var stdout bytes.Buffer
	opts.Stdout = &stdout
	opts.Stderr = &bytes.Buffer{}
	root.SetArgs(argv)
	if err := root.ExecuteContext(ctx); err != nil {
		var exitErr cli.ExitError
		if errors.As(err, &exitErr) && exitErr.Message != "" {
			return "", errors.New(exitErr.Message)
		}
		return "", err
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}
