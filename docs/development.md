# Development

## Prerequisites

- Go 1.26+ (see `go.mod` for the exact version)
- A working C compiler (for `creack/pty` — not needed on Linux with `golang.org/x/sys`)

## Quick start

```sh
# Build
go build ./cmd/relay-mcp

# Run (stdio transport — the MCP client drives it)
./relay-mcp

# Test
go test ./...

# With race detector
go test -race -count=1 ./...
```

## Project layout

```
cmd/relay-mcp/     # Binary entry point
internal/
├── server/        # MCP protocol layer
└── session/       # Terminal session layer
```

Each package owns one responsibility. Namespace parents (e.g. `internal/session/`) contain only docs; sub-packages are the real code.

## Testing

- Unit tests live next to the code they test (`*_test.go` in the same package).
- Integration tests that spawn real PTY sessions are in `internal/session/session/`.
- E2E tests that exercise the full MCP server are in `internal/server/server/`.
- Run with `go test -race -shuffle=on -count=1 ./...` for CI-grade coverage.

## Before committing

```sh
go vet ./...
go test -race -shuffle=on -count=1 ./...
```

## Commit style

Conventional Commits, no scope parentheticals, no AI attribution:

```
feat: add run_command one-shot tool
fix: prevent race on session close
chore: bump creack/pty to v1.1.22
```

## Related

- [architecture.md](./architecture.md) — system architecture
