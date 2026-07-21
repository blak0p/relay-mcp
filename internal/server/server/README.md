# server

MCP server wiring for `relay-mcp`.

`NewServer(reg)` builds a `*mcpserver.MCPServer` from `mark3labs/mcp-go` with
every relay-mcp tool registered. It is the single assembly point between the
session core (`internal/session/...`) and the MCP protocol layer.

## Registration map

| Tool              | Name constant                   | Handler                       | Status    |
|-------------------|---------------------------------|-------------------------------|-----------|
| `create_terminal` | `description.CreateTerminalName`| `handler.New(reg)`             | v1        |
| `write_terminal`  | `description.WriteTerminalName`| `handler.NewWriteTerminalHandler(reg)` | v1 |
| `read_terminal`   | `description.ReadTerminalName` | `handler.NewReadTerminalHandler(reg)` | v1 |
| `send_control`    | (future)                        | (future)                      | planned   |
| `close_terminal`  | (future)                        | (future)                      | planned   |

Each tool's name, summary, and description come from the `description` package
(REQ-008: single source of truth for tool metadata). The handler receives the
shared `*registry.Registry` by injection — the same instance must be passed to
every handler so the single-session invariant holds.

## Usage

```go
reg := registry.NewRegistry()
s, err := server.NewServer(reg)
if err != nil { /* ... */ }
if err := mcpserver.ServeStdio(s); err != nil { /* ... */ }
```

`cmd/relay-mcp/main.go` is the thin entry point that does exactly this and
wires SIGINT/SIGTERM to a cancellable context.

## Server identity

`NewServer` advertises:

- `ServerName` = `"relay-mcp"`
- `ServerVersion` = `"0.1.0"` (static in v1; will be wired to a build-time
  variable in a future release)

## Layout

```
description  ←  handler  ←  server  ←  cmd/relay-mcp
                               ↓
                    mcpserver (mark3labs/mcp-go)
```

See `internal/server/README.md` for the full namespace map.
