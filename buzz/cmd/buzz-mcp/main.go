// Command buzz-mcp is the sibling MCP server for buzz-cli: it exposes the
// CLI's operations as MCP tools over stdio, reusing buzz-cli's own
// internal/{nostr,client,config,cli} logic (see internal/mcp) rather than
// reimplementing protocol/event handling.
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	mcptools "buzz-cli/internal/mcp"
)

var version = "0.0.0-dev"

func main() {
	s := server.NewMCPServer("Buzz", version, server.WithToolCapabilities(false))
	mcptools.RegisterTools(s)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "buzz-mcp: MCP server error: %v\n", err)
		os.Exit(1)
	}
}
