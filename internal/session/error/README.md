# error

Typed sentinel errors for the session lifecycle.

## Purpose

`error` (package name `serror`) defines the sentinel errors used across the
[`session`](..) namespace, plus the helper that carries the existing session
id through `ErrSessionAlreadyExists`. The handler layer (PR2) maps each
sentinel to a distinct JSON-RPC error code so clients can branch on errors
without string matching.

### Why `serror` and not `error`

The package folder is `error/` but the package name is `serror`. A package
named `error` would shadow the builtin predeclared `error` type within its
own files (the `existingSessionError` wrapper references the builtin in
method signatures). External callers import the package as
`github.com/blak0p/relay-mcp/internal/session/error` and refer to its
exported identifiers directly — the package name only matters inside the
folder.

## Sentinels

| Error | Trigger | JSON-RPC code (PR2) |
|-------|---------|---------------------|
| `ErrSessionAlreadyExists` | `Registry.Put` when a session is already registered. Carries the existing id. | -32001 |
| `ErrBashNotFound` | `exec.LookPath("bash")` fails. No fallback shell. | -32002 |
| `ErrSpawnFailed` | `pty.StartWithSize` or `cmd.Start` fails. | -32003 |
| `ErrSessionNotFound` | `Registry.Get` on an empty registry. | -32004 |

## Exported API

| Identifier | Kind | Description |
|------------|------|-------------|
| `ErrSessionAlreadyExists` | sentinel | A session is already active; call `close_terminal` first. |
| `ErrBashNotFound` | sentinel | `bash` not found in `PATH`. |
| `ErrSpawnFailed` | sentinel | PTY or process start failed. |
| `ErrSessionNotFound` | sentinel | No session registered / id does not match. |
| `ExistingSessionID(err) string` | func | Extracts the conflicting id from an `ErrSessionAlreadyExists` error; `""` if the error is not an existing-session error. |
| `NewExistingSessionError(id) error` | func | Builds the wrapped error carrying the id. Exported for the sibling `registry` package. |

## Usage

```go
import (
    "errors"
    "github.com/blak0p/relay-mcp/internal/session/error"
)

if err := reg.Put(s); err != nil {
    if errors.Is(err, serror.ErrSessionAlreadyExists) {
        id := serror.ExistingSessionID(err)
        // return JSON-RPC -32001 with {"existingId": id} in error.data
    }
    // ...
}

cur, err := reg.Get()
if errors.Is(err, serror.ErrSessionNotFound) {
    // return JSON-RPC -32004
}
```

`errors.Is` works through the wrapper because `existingSessionError.Unwrap`
returns the inner sentinel. Use `ExistingSessionID` only after `errors.Is`
has confirmed the kind.

## Related packages

- [`registry`](../registry/README.md) — the producer of
  `ErrSessionAlreadyExists` and `ErrSessionNotFound`.
- [`session`](../session/README.md) — the lifecycle that produces
  `ErrBashNotFound` and `ErrSpawnFailed` at spawn time (wired in PR2's
  handler).

## Testing

```bash
go test ./internal/session/error/ -count=1 -race
```