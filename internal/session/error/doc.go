// Package serror defines the typed sentinel errors used across the session
// namespace, plus the helpers that carry the existing session id through
// ErrSessionAlreadyExists. The handler layer maps each sentinel to a distinct
// JSON-RPC error code (see the package README for the full mapping).
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
