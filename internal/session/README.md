# session

Namespace parent for the session lifecycle in relay-mcp.

## Purpose

The `session` namespace groups every concern related to a PTY-backed bash
session: the core domain types, the thread-safe single-session store, the
liveness primitive, and the typed sentinel errors. The parent itself holds
no Go code — only this README and a `doc.go` package comment. The real units
of code are the sub-packages, each independently importable.

## Sub-packages

```
internal/session/
├── session/    core types: Session, SessionState, New(), Close(), ReconcileState()
├── registry/   thread-safe single-session store: NewRegistry(), Put(), Get()
├── liveness/   cheap liveness check: IsAlive(pid) via signal-0 ping
└── error/      typed sentinel errors + ExistingSessionID() helper (package serror)
```

### When to use which

| Sub-package | Use when |
|-------------|----------|
| [`session/`](session/README.md) | You need the `Session` struct, `SessionState`, or the `New`/`Close`/`ReconcileState` lifecycle. |
| [`registry/`](registry/README.md) | You need to store or retrieve the active session under the single-session invariant. |
| [`liveness/`](liveness/README.md) | You need to ask the kernel whether a pid is still alive, without reaping it. |
| [`error/`](error/README.md) | You need to branch on a session failure mode or extract the existing session id from an error. |

## Design notes

- **Single-active-session invariant.** MCP over stdio serves exactly one
  client, so the registry holds one slot. Enforced at the registry level,
  not in the handler.
- **Lazy liveness.** No goroutine watches the bash process. The stored
  state stays `running` until a session operation triggers
  `ReconcileState()`. `Registry.Get()` reconciles before returning, so
  every read sees up-to-date state.
- **Signal-0 ping.** `liveness.IsAlive` uses `unix.Kill(pid, 0)` — no
  reaping, no signaling, no side effects.
- **No MCP wiring here.** This namespace is pure domain logic. The
  handler/server wiring lives in `internal/server` (PR2).

## Testing

Strict TDD. Unit tests cover every sub-package; an integration test in
`session/` spawns a real bash inside a PTY and verifies the 100x30 size
contract (skips when `/bin/bash` is unavailable).

```bash
go test ./internal/session/... -count=1 -race
```

## Out of scope

- `write_terminal`, `read_terminal`, `send_control`, `close_terminal`
- Proactive process monitoring (a goroutine waiting on bash)
- Shell selection, env config, working directory
- PTY resize
- Windows support