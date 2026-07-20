# liveness

Cheap process-existence checks for PTY-backed sessions.

## Purpose

`liveness.IsAlive(pid)` reports whether a process with the given pid is still
running, using a signal-0 ping (`unix.Kill(pid, 0)`). It is the primitive
that powers the lazy liveness reconciliation in
[`session.ReconcileState`](../session/README.md#liveness-model).

## Exported API

| Identifier | Kind | Description |
|------------|------|-------------|
| `IsAlive(pid int) bool` | func | True if a process with `pid` exists. Uses `unix.Kill(pid, 0)`. |

## Usage

```go
import "github.com/blak0p/relay-mcp/internal/session/liveness"

if liveness.IsAlive(pid) {
    // process still running
} else {
    // process is dead (or pid is invalid / not ours)
}
```

## Why signal-0

`unix.Kill(pid, 0)` is the standard Unix "signal-0 ping": the kernel checks
whether the pid exists and whether the caller may signal it, without
delivering any signal. It has three properties that make it the right
primitive for liveness checks:

- **No side effects.** Unlike `wait4`/`waitpid`, it does not reap the child
  or change its state. Reaping here would steal the exit status from whoever
  owns the process.
- **Cheap.** A single syscall, no allocation.
- **Composable.** The caller decides what to do with the answer. `IsAlive`
  does not mutate the `Session` or its `Cmd`.

## Limitations

- **PID reuse.** `IsAlive` only answers "does *some* process with this pid
  exist", not "is it *our* process". For relay-mcp's single-session,
  short-lived bash this is acceptable; for long-lived multi-session servers
  a pidfd-based check would be safer.
- **Unix only.** Uses `golang.org/x/sys/unix.Kill`. Windows is not a v1
  target.
- **No exit code.** `IsAlive` does not return *how* the process died. The
  exit code is read from `cmd.ProcessState` in
  [`session.classifyExit`](../session/README.md).

## Related packages

- [`session`](../session/README.md) — calls `IsAlive` from `ReconcileState`.
- [`registry`](../registry/README.md) — triggers `ReconcileState` on `Get`.

## Testing

```bash
go test ./internal/session/liveness/ -count=1 -race
```