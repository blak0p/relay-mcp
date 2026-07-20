# session

Core domain types for PTY-backed bash sessions in relay-mcp.

## Purpose

This package owns the `Session` struct, its lifecycle (`New`, `Close`), the
`SessionState` enum, and the lazy liveness reconciliation (`ReconcileState`).
It is the foundation of the session namespace: sibling sub-packages
([registry](../registry/README.md), [liveness](../liveness/README.md),
[error](../error/README.md)) build on these types.

The package does **not** speak MCP, does not own the registry, and does not
own the liveness primitive. It is pure domain logic.

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
    // unexported: closed bool, mu sync.Mutex
}
```

A `Session` is created via `New(cmd, pty)` and owns the PTY master file
descriptor. The caller is responsible for starting the process (typically
`creack/pty.StartWithSize`).

### `SessionState`

Three states:

| State | Meaning |
|-------|---------|
| `StateRunning` | At creation. Stays `running` after bash dies until `ReconcileState` runs. |
| `StateExited` | Bash exited with code 0, or exit info is unavailable. |
| `StateError` | Bash died from a signal or a non-zero exit code. |

## Exported API

| Identifier | Kind | Description |
|------------|------|-------------|
| `Session` | struct | One PTY-backed bash process. |
| `SessionState` | type | Lifecycle state enum (`string`). |
| `StateRunning`, `StateExited`, `StateError` | consts | The three states. |
| `New(cmd, pty) *Session` | func | Constructor; generates id via `idgen.New()`, sets state to `StateRunning`. Does not start the process. |
| `(s *Session) Close() error` | method | Closes the PTY master FD. Idempotent. Does not kill or wait on the process. |
| `(s *Session) ReconcileState()` | method | Lazy liveness: if `StateRunning` but the pid is dead, flip to `StateExited`/`StateError`. Safe for concurrent use. |

## Usage

```go
import (
    "os/exec"
    "github.com/creack/pty"
    "github.com/blak0p/relay-mcp/internal/session/session"
)

cmd := exec.Command("bash", "-i")
win := &pty.Winsize{Rows: 30, Cols: 100}
ptmx, err := pty.StartWithSize(cmd, win)
if err != nil { /* ... */ }
s := session.New(cmd, ptmx)
// s.PID, s.ID, s.State are populated; cmd is already started by pty.StartWithSize
defer s.Close()
```

## Liveness model

The package does **not** proactively monitor the bash process — no goroutine
waits on it. The stored state stays `running` even after bash dies, until a
session operation calls `ReconcileState()`. `ReconcileState` uses
[`liveness.IsAlive`](../liveness/README.md) (`unix.Kill(pid, 0)`) to ask the
kernel whether the pid still exists, with no side effects.

This is the lazy liveness contract: clients learn about state changes only
when they trigger a session operation. Proactive monitoring is explicitly
deferred to the `read_terminal` SDD.

## Related packages

- [`registry`](../registry/README.md) — stores the active session and calls
  `ReconcileState` on every `Get`.
- [`liveness`](../liveness/README.md) — the `IsAlive(pid)` primitive used by
  `ReconcileState`.
- [`error`](../error/README.md) — sentinel errors for the session lifecycle.
- [`idgen`](../../idgen/README.md) — generates the `term_`-prefixed id.

## Testing

```bash
go test ./internal/session/session/ -count=1 -race
go test ./internal/session/session/ -run TestIntegration -v   # real bash
```

The integration test spawns a real bash inside a PTY and verifies the 100x30
size contract; it skips via `t.Skip` when `/bin/bash` is unavailable.