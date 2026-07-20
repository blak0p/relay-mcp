// Package handler implements the MCP tool handlers for relay-mcp.
//
// Each tool handler is a server.ToolHandlerFunc returned by a constructor
// (New, and in the future NewWriteTerminalHandler, etc.) that takes the
// shared session Registry by injection. Handlers are responsible for:
//
//   - performing the tool's side effect (spawning a PTY, writing input, etc.),
//   - mapping internal sentinel errors (from package serror) to the stable
//     error codes defined in the design (-32001..-32004), and
//   - returning a mcp.CallToolResult whose Content is a JSON string carrying
//     either the success payload or the error code/message/data triple.
//
// Tool execution failures are surfaced as CallToolResult{IsError:true} with
// the error payload in the text content. This matches how the MCP spec and
// mark3labs/mcp-go model tool-level errors: protocol-level JSON-RPC error
// responses are reserved for transport/protocol problems, while tool failures
// stay inside the result object so the model sees them in its context window.
//
// See the package README.md for the error code table and usage examples.
package handler
