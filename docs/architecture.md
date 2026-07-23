# Architecture

relay-mcp is a Go binary that implements the [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server over stdio. It exposes terminal sessions to AI agents as MCP tools.

## Layers

```
cmd/relay-mcp/          # Binary entry point — wiring only
internal/
├── server/             # MCP protocol layer
│   ├── server/         #   Server construction, tool registration
│   ├── handler/        #   Tool handler functions
│   └── description/    #   Tool metadata (single source of truth)
└── session/            # Terminal session layer
    ├── session/        #   Core session type, lifecycle
    ├── registry/       #   Session registry (CRUD)
    ├── liveness/       #   Process health checks
    ├── output/         #   Output broker (ring buffer + consumers)
    └── error/          #   Typed sentinel errors
```

## Data flow

```
AI agent
  │ create_terminal({ command: "bash" })
  ▼
relay-mcp ──► creack/pty ──► process (bash / python / ...)
  │                              │
  │  write_terminal              │ stdout
  │  "echo hola\n" ────────────► │
  │                              │
  │  read_terminal ◄──────────── │ "hola\n"
  │                              │
  │  send_control("ctrl+c") ───► │ SIGINT
  │                              │
  │  close_terminal ────────────► │ kill
```

## Key design decisions

- **5 tools, well made** — not 27 like Forge. Core tools cover 90% of real agent interaction.
- **Static binary** — Go compiles to a single binary, zero runtime dependencies.
- **Ring buffer** — circular byte buffer with multi-consumer cursors. No data loss on slow readers.
- **Process group isolation** — each session runs in its own process group. Close kills the whole tree.
- **No ANSI emulator** — raw PTY output. `read_screen` (viewport rendering) is a future phase.

## MCP transport

The server speaks JSON-RPC 2.0 over stdin/stdout. The MCP client (Claude Desktop, Claude Code, etc.) drives the conversation.

## Related

- [development.md](./development.md) — building, testing, contributing
