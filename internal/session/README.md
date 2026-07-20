# session

Core domain model for PTY-backed bash sessions in relay-mcp.

## Purpose

This package owns the lifecycle of a terminal session: spawning, identifying,
tracking, and reconciling the state of a bash process attached to a
pseudo-terminal (PTY). It is the foundation for the `create_terminal` MCP tool
and the future `write_terminal`, `read_terminal`, `send_control`, and
`close_terminal` tools.

The package does NOT speak MCP — that wiring lives in `internal/server` (PR2).
This is pure domain logic, fully testable in isolation.

## Types

### `Session`

```go
type Session struct {
    ID        string       // "term_" + 16 hex chars (idgen.New())
    PTY       *os.File     // master end of the PTY
    Cmd       *exec.Cmd    // bash process handle
    PID       int          // cached for liveness checks
    StartedAt time.Time
    State     SessionState // running | exited | error
}
```

A `Session` is created via `New(cmd, pty)` and owns the PTY master file
descriptor. The caller is responsible for starting the process (typically
`creack/pty.StartWithSize`).

### `SessionState`

Three states: `running` (at creation and until liveness is reconciled),
`exited` (bash exited with code 0 or state is unknown), `error` (bash died
from a signal or non-zero exit code).

### `Registry`

Thread-safe single-session store. MCP over stdio serves exactly one client,
so a single slot is sufficient (see design decision: "Registry scope").
Enforces the single-session invariant at the registry level — not in the
handler.

```go
reg := NewRegistry()
err := reg.Put(session)            // ErrSessionAlreadyExists if occupied
s, err := reg.Get()                // ErrSessionNotFound if empty; reconciles state first
id := ExistingSessionID(err)      // extract existing id from the error
```

## Liveness model (REQ-007 + REQ-009)

The package does **not** proactively monitor the bash process — no goroutine
waits on it. The stored state stays `running` even after bash dies, until a
session operation calls `reconcileState()`.

`Registry.Get()` runs `reconcileState()` before returning, so every session
access returns a reconciled state. This is the lazy liveness contract:
clients learn about state changes only when they trigger a session operation.
Proactive monitoring is explicitly deferred to the `read_terminal` SDD.

`reconcileState()` uses `unix.Kill(pid, 0)` — the standard Unix signal-0
"ping" that checks existence without side effects (no reaping, unlike
`wait4`).

## Error taxonomy

Typed sentinel errors so the MCP handler (PR2) can map each failure to a
distinct JSON-RPC error code without string matching:

| Error | Trigger | JSON-RPC code (PR2) |
|-------|---------|---------------------|
| `ErrSessionAlreadyExists` | a session is already registered | -32001 |
| `ErrBashNotFound` | `exec.LookPath("bash")` fails | -32002 |
| `ErrSpawnFailed` | `pty.StartWithSize` or `cmd.Start` fails | -32003 |
| `ErrSessionNotFound` | `registry.Get()` on empty registry | -32004 |

`ErrSessionAlreadyExists` carries the existing session id; extract it via
`ExistingSessionID(err)`.

## Testing

Strict TDD. Unit tests cover id generation, sentinel errors, the Session
constructor, `Close()` idempotency, the registry guard, liveness, and
`reconcileState`. An integration test spawns a real bash inside a PTY and
verifies the 100x30 size contract; it skips via `t.Skip` when `/bin/bash` is
unavailable.

Run:

```bash
go test ./internal/session/ -count=1 -race
go test ./internal/session/ -run TestIntegration -v   # real bash
```

## Out of scope (handled by other SDDs)

- `write_terminal`, `read_terminal`, `send_control`, `close_terminal`
- Proactive process monitoring (a goroutine waiting on bash)
- Shell selection, env config, working directory
- PTY resize
- Windows support (`creack/pty` is Unix-only)