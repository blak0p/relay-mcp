# cmd/relay-mcp

Entry-point binary for the relay-mcp MCP server. This is the only `main`
package in the repository; it is the wiring layer, not the logic layer.

## What lives here

- `main.go` — the binary. It builds the session registry, builds the MCP
  server via `internal/server/server.NewServer`, and serves it over stdio.
  A signal handler turns SIGINT/SIGTERM into a clean context cancellation.

## What does NOT live here

- MCP tool definitions → `internal/server/handler`
- Tool descriptions (single source of truth) → `internal/server/description`
- Server construction and registration → `internal/server/server`
- Session types, registry, liveness, errors → `internal/session/*`

If you find yourself adding business logic to this package, it belongs
somewhere else. The whole point of `cmd/` is to be a thin shim that
`go build` can turn into a binary.

## How to run

```sh
go run ./cmd/relay-mcp
```

The MCP client drives the server over stdin/stdout — there are no CLI
flags. See `PRODUCT-BRIEF.md` at the repo root for the protocol shape.

## Documentation convention

`cmd/relay-mcp/` is a documented exception to the docs-pair rule
(see `CONVENTIONS.md` §3). The package doc comment lives at the top of
`main.go` (idiomatic Go for `package main`); this `README.md` is the
human-facing half of the pair. There is no `doc.go`.
