# registry

Thread-safe single-session store for relay-mcp.

## Purpose

`registry` enforces the single-active-session invariant: at most one
`Session` may be registered at a time. MCP over stdio serves exactly one
client, so a single slot is sufficient — a map keyed by client id would add
complexity with zero benefit.

The store also performs lazy liveness reconciliation on `Get`: before
returning the session, it calls `ReconcileState()` so callers always see
up-to-date state.

## Exported API

| Identifier | Kind | Description |
|------------|------|-------------|
| `Registry` | struct | Thread-safe single-session store. All methods are safe for concurrent use. |
| `NewRegistry() *Registry` | func | Returns an empty registry. |
| `(r *Registry) Put(s *session.Session) error` | method | Stores `s` as the active session. Returns `ErrSessionAlreadyExists` (wrapped with the existing id) if a session is already registered; the existing session is **not** replaced. |
| `(r *Registry) Get() (*session.Session, error)` | method | Returns the active session after running `ReconcileState`. Returns `ErrSessionNotFound` if empty. |

## Usage

```go
import (
    "github.com/blak0p/relay-mcp/internal/session/registry"
    "github.com/blak0p/relay-mcp/internal/session/session"
    "github.com/blak0p/relay-mcp/internal/session/error"
)

reg := registry.NewRegistry()

// Register a session.
if err := reg.Put(s); err != nil {
    if errors.Is(err, serror.ErrSessionAlreadyExists) {
        id := serror.ExistingSessionID(err) // the conflicting id
        // surface id to the client in the JSON-RPC error data
    }
}

// Retrieve the active session (reconciles state first).
cur, err := reg.Get()
if err != nil { /* ErrSessionNotFound */ }
// cur.State is up to date
```

## Why single-session

MCP over stdio serves exactly one client per process. A multi-session map
would imply multi-client semantics that the transport does not support, and
would require eviction policy, TTL handling, and id routing — all of which
are out of scope for v1. The single-slot store keeps the invariant at the
registry level instead of scattering it across handlers.

## Liveness on Get

`Get` runs `ReconcileState()` on the stored session before returning. This
means every read reflects the real process state: if bash died since the last
access, the returned `State` is `exited` or `error`, not a stale `running`.
See [`session.ReconcileState`](../session/README.md#liveness-model) for the
mechanism.

## Related packages

- [`session`](../session/README.md) — the `Session` type stored here.
- [`error`](../error/README.md) — `ErrSessionAlreadyExists`,
  `ErrSessionNotFound`, and `ExistingSessionID(err)` to extract the
  conflicting id.
- [`liveness`](../liveness/README.md) — the primitive `ReconcileState` uses.

## Testing

```bash
go test ./internal/session/registry/ -count=1 -race
```