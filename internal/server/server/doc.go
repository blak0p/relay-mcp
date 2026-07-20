// Package server wires the MCP server from mark3labs/mcp-go with the
// relay-mcp tool handlers. NewServer is the single entry point: it builds
// an *mcpserver.MCPServer, registers every tool (name, summary, and
// description sourced from the description package — the single source of
// truth), and returns it ready for a transport to serve.
//
// See the package README.md for the registration map and how to serve the
// returned server over stdio from cmd/relay-mcp.
package server
