package server

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/blak0p/relay-mcp/internal/server/description"
	"github.com/blak0p/relay-mcp/internal/server/handler"
	"github.com/blak0p/relay-mcp/internal/session/registry"
)

// ServerName is the MCP server implementation name advertised in the
// initialize response. It identifies the product to MCP clients.
const ServerName = "relay-mcp"

// ServerVersion is the advertised MCP server version. v1 reports a static
// "0.1.0"; a future release will wire this to a build-time variable.
const ServerVersion = "0.1.0"

// NewServer builds a *mcpserver.MCPServer with the create_terminal tool
// registered. reg must be non-nil; the same registry instance must be
// shared with every handler (single-session invariant).
//
// The tool is registered with the name/summary/description sourced from the
// description package (REQ-008: single source of truth for tool metadata),
// and the handler built by handler.New(reg).
func NewServer(reg *registry.Registry) (*mcpserver.MCPServer, error) {
	if reg == nil {
		return nil, fmt.Errorf("server: NewServer requires a non-nil registry")
	}
	s := mcpserver.NewMCPServer(ServerName, ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	tool := mcp.NewTool(
		description.CreateTerminalName,
		mcp.WithDescription(description.CreateTerminalDescription),
	)
	s.AddTool(tool, handler.New(reg))

	// write_terminal: inject raw bytes into the active session's PTY. The
	// schema declares one required string parameter "data" (REQ-WT-008).
	// Two explicit AddTool calls — obvious, grep-friendly, zero abstraction
	// overhead for 2 tools (see design: "two explicit calls over a loop").
	writeTool := mcp.NewTool(
		description.WriteTerminalName,
		mcp.WithDescription(description.WriteTerminalDescription),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("Raw bytes to inject into the terminal session (max 1 MiB). No auto-Enter — include \\n if you want to submit."),
		),
	)
	s.AddTool(writeTool, handler.NewWriteTerminalHandler(reg))

	return s, nil
}
