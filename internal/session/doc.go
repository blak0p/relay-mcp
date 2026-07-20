// Package session is the namespace parent for the session lifecycle in
// relay-mcp. It groups the sub-packages that own session concerns:
//
//   - session:  core types (Session, SessionState, lifecycle, reconciliation)
//   - registry: thread-safe single-session store
//   - liveness: cheap liveness check (IsAlive) via signal-0 ping
//   - error:    typed sentinel errors for session operations
//
// Sub-packages are independently importable. The parent itself contains only
// this doc comment — no Go code lives here.
package session
