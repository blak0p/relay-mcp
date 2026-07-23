# relay-mcp

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![CI](https://github.com/blak0p/relay-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/blak0p/relay-mcp/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Interactive terminal MCP server for AI agents.**

Relay lets AI agents spawn real PTY sessions — bash, python, lazygit, whatever — and interact with them through MCP tools. The agent delegates commands, you execute them in a real terminal, and the agent reads the output. Like a relay.

## Tools

| Tool | What it does |
|---|---|
| `create_terminal` | Spawn a new PTY session (bash by default) |
| `write_terminal` | Write input to the session (no auto-Enter — include `\n`) |
| `read_terminal` | Read output incrementally (stream, snapshot, or drain) |
| `send_control` | Send control sequences: Ctrl+C, arrows, Tab, etc. |
| `close_terminal` | Kill the session and free its resources |

5 tools, well made. Covers 90% of real agent-terminal interaction.

## Quick start

```sh
go install github.com/blak0p/relay-mcp/cmd/relay-mcp@latest
relay-mcp
```

The MCP client drives the server over stdin/stdout — no flags, no config.

Or build from source:

```sh
git clone https://github.com/blak0p/relay-mcp.git
cd relay-mcp
go build ./cmd/relay-mcp
./relay-mcp
```

## Architecture

```
AI agent ──► relay-mcp ──► creack/pty ──► process (bash / python / ...)
                 │
                 ├── create_terminal     spawn PTY
                 ├── write_terminal      send input
                 ├── read_terminal       read output
                 ├── send_control        Ctrl+C, arrows, Tab
                 └── close_terminal      kill session
```

Each session runs in an isolated process group. Close kills the whole process tree. Output is buffered in a ring buffer — no data loss on slow readers.

See [docs/architecture.md](docs/architecture.md) for the full picture.

## Configuration

None. relay-mcp is a stdio MCP server with zero configuration. Point your MCP client at the binary and it works.

## Documentation

- [docs/architecture.md](docs/architecture.md) — system architecture and design decisions
- [docs/development.md](docs/development.md) — building, testing, contributing

## License

MIT. See [LICENSE](LICENSE).
