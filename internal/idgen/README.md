# idgen

Unique session id generation for relay-mcp terminal sessions.

## Purpose

`idgen.New()` returns a unique, unpredictable id of the form `term_` followed
by 16 lowercase hex characters (8 bytes from `crypto/rand`).

- **Zero dependencies** — stdlib only (`crypto/rand`, `encoding/hex`).
- **Collision-resistant** — 2^64 id space; for a single-session server the
  collision probability is negligible.
- **Short** — 21 chars total, easy to read in logs and JSON-RPC responses.

## Usage

```go
import "github.com/blak0p/relay-mcp/internal/idgen"

id := idgen.New() // "term_a1b2c3d4e5f6a7b8"
```

## Format

```
term_ + [0-9a-f]{16}
```

The `term_` prefix makes ids greppable in logs and unambiguous in MCP
responses. The 16 hex chars come from 8 random bytes encoded with
`encoding/hex`, so the output is always lowercase and exactly 16 chars.

## Failure mode

`New()` panics if `crypto/rand` is unavailable. This is intentional: silently
returning a duplicate or weak id would be worse than failing loudly. On
Linux kernel 5.6+ `getrandom()` is non-blocking, so this path is effectively
unreachable in production.