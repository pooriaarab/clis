package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestRegisterToolsUniqueNames is the runnable check for the tool table:
// every toolDef must produce a distinct, non-empty MCP tool name and a
// schema mcp-go accepts, or RegisterTools would silently drop or clobber a
// tool at server start.
func TestRegisterToolsUniqueNames(t *testing.T) {
	s := server.NewMCPServer("test", "0.0.0-test")
	RegisterTools(s)
	seen := map[string]bool{}
	for _, def := range toolDefs {
		if strings.TrimSpace(def.Name) == "" {
			t.Fatalf("toolDef with path %v has an empty Name", def.Path)
		}
		if seen[def.Name] {
			t.Fatalf("duplicate tool name %q", def.Name)
		}
		seen[def.Name] = true
		if s.GetTool(def.Name) == nil {
			t.Errorf("tool %q was not registered", def.Name)
		}
	}
}

// TestHandlerRoundTripsThroughRealCLI runs a real (read-only, no-network)
// buzz-cli command end to end through the generic handler: it proves the
// arg-table -> argv -> cli.NewRootCommand() plumbing actually produces the
// CLI's own JSON output, not just that it compiles.
func TestHandlerRoundTripsThroughRealCLI(t *testing.T) {
	def := findToolDef(t, "invite_list")
	handler := makeHandler(def)
	cfgPath := filepath.Join(t.TempDir(), "missing-config.toml")

	result, err := handler(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      def.Name,
			Arguments: map[string]any{"config": cfgPath},
		},
	})
	if err != nil {
		t.Fatalf("handler returned a protocol-level error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler reported a tool error: %+v", result.Content)
	}
	text := resultText(t, result)
	if strings.TrimSpace(text) != "[]" {
		t.Errorf("invite_list on a fresh config = %q, want %q", text, "[]")
	}
}

// TestHandlerSurfacesCLIValidationErrors checks that a required-flag miss
// comes back as a tool error carrying the CLI's own message, not a generic
// failure — the whole point of running the real command tree instead of a
// hand-rolled validator.
func TestHandlerSurfacesCLIValidationErrors(t *testing.T) {
	def := findToolDef(t, "channels_get")
	handler := makeHandler(def)

	result, err := handler(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Name: def.Name, Arguments: map[string]any{}},
	})
	if err != nil {
		t.Fatalf("handler returned a protocol-level error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected a tool error for a missing required --channel flag, got: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "channel") {
		t.Errorf("error message %q does not mention the missing flag", resultText(t, result))
	}
}

// TestHandlerRequiresPositionalArg checks the positional-arg guard for
// commands like `agents get <name|pubkey>` that take one CLI positional
// instead of a flag.
func TestHandlerRequiresPositionalArg(t *testing.T) {
	def := findToolDef(t, "agents_get")
	handler := makeHandler(def)

	result, err := handler(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{Name: def.Name, Arguments: map[string]any{}},
	})
	if err != nil {
		t.Fatalf("handler returned a protocol-level error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected a tool error for a missing positional target, got: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "target") {
		t.Errorf("error message %q does not mention the missing argument", resultText(t, result))
	}
}

func findToolDef(t *testing.T, name string) toolDef {
	t.Helper()
	for _, def := range toolDefs {
		if def.Name == name {
			return def
		}
	}
	t.Fatalf("no toolDef named %q", name)
	return toolDef{}
}

func resultText(t *testing.T, result *mcplib.CallToolResult) string {
	t.Helper()
	var out strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(mcplib.TextContent); ok {
			out.WriteString(tc.Text)
		}
	}
	return out.String()
}
