package mcp

import (
	"context"
	"strconv"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// makeHandler returns the generic MCP tool handler for def: it maps the
// call's arguments back onto `--<flag>` CLI args (plus the command's
// positional arg, if it has one) and runs them through runCLI. The result
// text is exactly buzz-cli's own stdout for that invocation.
func makeHandler(def toolDef) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		args := req.GetArguments()

		argv := append([]string{}, def.Path...)
		for _, f := range def.Flags {
			argv = append(argv, flagArgs(f, args)...)
		}
		for _, g := range globalFlags {
			argv = append(argv, flagArgs(g, args)...)
		}
		if def.Positional != "" {
			value, _ := args[def.Positional].(string)
			value = strings.TrimSpace(value)
			if value == "" {
				if def.PositionalRequired {
					return mcplib.NewToolResultError("missing required argument: " + def.Positional), nil
				}
			} else {
				argv = append(argv, value)
			}
		}

		out, err := runCLI(ctx, argv)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		if out == "" {
			out = "{}"
		}
		return mcplib.NewToolResultText(out), nil
	}
}

// flagArgs renders one flagDef as `--name value` (or `--name=bool`) CLI
// args from the MCP call's argument map, or nil when the caller didn't set
// it — buzz-cli's own flag default then applies, same as omitting the flag
// on the command line.
func flagArgs(f flagDef, args map[string]any) []string {
	v, ok := args[f.Name]
	if !ok || v == nil {
		return nil
	}
	flag := "--" + f.Name
	switch f.Kind {
	case flagBool:
		b, _ := v.(bool)
		return []string{flag + "=" + strconv.FormatBool(b)}
	case flagInt:
		switch n := v.(type) {
		case float64:
			return []string{flag, strconv.FormatInt(int64(n), 10)}
		case string:
			if strings.TrimSpace(n) == "" {
				return nil
			}
			return []string{flag, n}
		default:
			return nil
		}
	case flagStringSlice:
		arr, ok := v.([]any)
		if !ok {
			return nil
		}
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) == 0 {
			return nil
		}
		return []string{flag, strings.Join(parts, ",")}
	default: // flagString
		s, _ := v.(string)
		return []string{flag, s}
	}
}
