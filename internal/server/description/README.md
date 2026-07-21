# description

Tool description constants for the `relay-mcp` MCP server.

This package is the single source of truth for the `name`, `summary`, and
`description` strings that `mark3labs/mcp-go` sends to clients in the tool
manifest. Centralising them here means a wording change is a one-file edit,
and the `handler` and `server` packages consume the same constants without
coupling to each other.

## Constants

| Constant                       | Value       | Used by                         |
|--------------------------------|-------------|---------------------------------|
| `CreateTerminalName`           | `"create_terminal"` | `server` (tool registration), `handler` (result shaping) |
| `CreateTerminalSummary`        | one-line summary    | `server` (tool registration) |
| `CreateTerminalDescription`    | full description    | `server` (tool registration) |
| `WriteTerminalName`            | `"write_terminal"`  | `server` (tool registration) |
| `WriteTerminalSummary`         | one-line summary    | `server` (tool registration) |
| `WriteTerminalDescription`     | full description (states 1 MiB cap + raw-byte/no-auto-Enter contract) | `server` (tool registration) |
| `ReadTerminalName`             | `"read_terminal"`   | `server` (tool registration), `handler` (request binding) |
| `ReadTerminalSummary`          | one-line summary    | `server` (tool registration) |
| `ReadTerminalDescription`      | full description (states default progress streaming and bounded polling alternatives) | `server` (tool registration) |

The remaining two tools (`send_control`, `close_terminal`) will add their own
constant blocks here in follow-up SDDs.

## Usage

```go
import (
    "github.com/blak0p/relay-mcp/internal/server/description"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

tool := mcp.NewTool(
    description.CreateTerminalName,
    mcp.WithDescription(description.CreateTerminalDescription),
)
s.AddTool(tool, handler)
```

## Layout

This package imports nothing from the rest of the project — it is a leaf in
the import graph, which keeps the dependency direction clean:

```
description  ←  handler  ←  server  ←  cmd/relay-mcp
```

See `internal/server/README.md` for the full namespace map.
