// Package server is the namespace parent for relay-mcp's MCP wiring layer.
//
// It contains no executable code itself; the real packages live in its
// sub-directories:
//
//   - description: tool name/summary/description constants (single source of truth)
//   - handler:     MCP tool handlers and error mapping
//   - server:      MCP server setup and tool registration
//
// See the package README.md for the full map and the import direction between
// the sub-packages.
package server
