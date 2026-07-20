# handler

MCP tool handlers for `relay-mcp`.

Each handler is a `server.ToolHandlerFunc` built by a constructor that takes
the shared session `Registry` by injection. The handler does the tool's side
effect, maps internal sentinel errors to stable error codes, and returns a
`mcp.CallToolResult` whose text content is a JSON string — either the success
payload or an `{code, message, data}` error envelope.

## `create_terminal`

`New(reg, opts...)` returns the `create_terminal` handler. It:

1. Rejects the call with `-32001` if a session is already active (the
   existing session id travels in `data.existing_id`).
2. Resolves `bash` in `PATH` — no fallback shell. Missing bash → `-32002`.
3. Spawns `bash -i` in a 100x30 PTY via `creack/pty.StartWithSize`. Spawn
   failure → `-32003`.
4. Builds a `session.Session`, stores it in the registry, and returns
   `{id, state:"running", started_at}`.

```go
h := handler.New(reg)
s.AddTool(tool, h)
```

### Error code table

| Code    | Constant                  | Trigger                                        | `data`                |
|---------|---------------------------|------------------------------------------------|-----------------------|
| `-32001`| session_already_exists    | A session is already registered                | `{existing_id}`       |
| `-32002`| bash_not_found            | `exec.LookPath("bash")` fails                 | —                     |
| `-32003`| spawn_failed              | `pty.StartWithSize` or registry Put fails     | — (or error text)     |
| `-32004`| session_not_found         | reserved for the other tools (close/read/...) | —                     |

### Why errors travel inside `CallToolResult`, not as JSON-RPC error responses

`mark3labs/mcp-go` turns any non-nil Go error from a tool handler into a
JSON-RPC error response with the fixed `INTERNAL_ERROR` code (-32603); the
`requestError` type that carries custom codes is unexported. The MCP spec
itself models tool execution failures as `CallToolResult{IsError:true}` —
protocol-level JSON-RPC errors are for transport/protocol problems, while
tool failures stay inside the result so the model sees them in its context
window. We follow that model and keep the stable code/message/data triple
in the result's text content, which is what the client branches on.

This is a deliberate **deviation from the original design** (which assumed
JSON-RPC error responses with custom codes); see the apply-progress notes.

## Test seams

`WithSpawner(fn)` replaces the default `pty.StartWithSize`-based spawner so
the spawn_failed path can be exercised without depending on a missing binary.
Tests use a real `registry.Registry` (no mock) so the handler ↔ registry
contract is exercised end-to-end.

## Layout

```
description  ←  handler  ←  server  ←  cmd/relay-mcp
                 ↓
              session/{registry,session,error}, idgen
```

See `internal/server/README.md` for the full namespace map.