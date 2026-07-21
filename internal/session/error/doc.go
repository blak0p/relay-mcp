// Package serror defines the typed sentinel errors used across the session
// namespace, plus the helpers that carry the existing session id through
// ErrSessionAlreadyExists. The handler layer maps each sentinel to a distinct
// JSON-RPC-style error code (see the package README for the full mapping):
//
//   -32001 ErrSessionAlreadyExists
//   -32002 ErrBashNotFound
//   -32003 ErrSpawnFailed
//   -32004 ErrSessionNotFound (create_terminal + write_terminal lookup failures)
//   -32005 ErrSessionNotAlive
//   -32006 ErrWriteTooLarge
//   -32007 ErrSessionClosed
//   -32602 ErrInvalidArgument (JSON-RPC invalid_params; missing/wrong-typed data)
//
// ErrSessionNotFound keeps -32004 to avoid breaking callers that branched on
// the pre-existing code; the write_terminal sentinels take -32005..-32007.
//
// The package name is intentionally "serror" (not "error") so it does not
// collide with the builtin predeclared error type within its own files.
// External callers import this package as
// github.com/blak0p/relay-mcp/internal/session/error and refer to its
// exported identifiers directly.
//
// ExistingSessionID(err) extracts the conflicting session id from an
// ErrSessionAlreadyExists error, so the handler can include it in the
// JSON-RPC error data.
package serror
