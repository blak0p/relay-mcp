# server

Namespace parent for relay-mcp's MCP wiring layer.

This directory holds no Go code of its own (only `doc.go` for the package
comment). The real packages live in its sub-directories and are each an
independently importable unit with a single responsibility.

## Sub-packages

| Sub-package    | Responsibility                                        | Imports                          |
|----------------|-------------------------------------------------------|----------------------------------|
| [`description`](./description) | Tool name/summary/description constants (single source of truth) | stdlib only — leaf of the graph |
| [`handler`](./handler)         | MCP tool handlers (`server.ToolHandlerFunc`) + error mapping      | `description`, `session/{registry,session,error}`, `idgen`, `creack/pty`, `mark3labs/mcp-go` |
| [`server`](./server)           | MCP server setup + tool registration                               | `handler`, `description`, `session/registry`, `mark3labs/mcp-go` |

## Import direction

```
description  ←  handler  ←  server  ←  cmd/relay-mcp
                 ↓                        ↓
              session/{registry,session,error}  mcpserver (mark3labs/mcp-go)
                 ↓
              idgen, creack/pty
```

No cycles. `description` is a leaf (stdlib only). `handler` consumes the
session core built in PR1 plus `description`. `server` consumes `handler` and
`description` and hands the assembled `*mcpserver.MCPServer` to the entry
point in `cmd/relay-mcp`.

## v1 status

PR2 (this namespace) wires the first tool, `create_terminal`. The remaining
four tools (`write_terminal`, `read_terminal`, `send_control`,
`close_terminal`) will add their own handlers to `handler/` and their own
constant blocks to `description/` in follow-up SDDs, following the same
layout rule (one sub-package per responsibility, grouped under this
namespace parent).

See `internal/session/README.md` for the session-lifecycle namespace that
backs this wiring layer.