package description

// Package description holds the exported constants for every MCP tool
// relay-mcp registers: name, short summary, and the human-readable
// description sent to the client in the tool manifest.
//
// Keeping all tool descriptions in a single package makes wording changes
// (e.g. rephrasing a tool summary) a one-file edit and lets the handler and
// server packages consume the same source of truth without coupling to each
// other.
//
// See the package README.md for the full list of constants and usage examples.
