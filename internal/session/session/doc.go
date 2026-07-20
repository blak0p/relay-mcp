// Package session owns the core domain types for a PTY-backed bash session:
// the Session struct, its lifecycle (New, Close), the SessionState enum, and
// the lazy liveness reconciliation performed on session access.
//
// This package does not speak MCP, does not own the registry, and does not
// own the liveness primitive — those concerns live in sibling sub-packages
// (registry, liveness) under the same session namespace. ReconcileState and
// classifyExit are methods of *Session and therefore live here, not in
// liveness.
//
// See the package README for usage examples and the list of exported
// identifiers.
package session
