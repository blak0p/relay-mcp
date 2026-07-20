package session

import "errors"

// Sentinel errors for the session lifecycle. Each failure mode maps to a
// distinct error so callers (the MCP handler in PR2) can branch on errors.Is
// without string matching.
//
// The design's JSON-RPC error codes (-32001..-32004) are wired in PR2's
// handler; PR1 only needs the typed errors themselves.
var (
	// ErrSessionAlreadyExists is returned when a session is already active.
	// Its message includes the existing session id once attached via
	// WithExistingID (see TASK-1.7).
	ErrSessionAlreadyExists = errors.New("a session is already active; call close_terminal first")

	// ErrBashNotFound is returned when bash cannot be resolved in PATH.
	// No fallback shell is used.
	ErrBashNotFound = errors.New("bash not found in PATH; install bash or set $SHELL")

	// ErrSpawnFailed is returned when the PTY or process cannot be started.
	ErrSpawnFailed = errors.New("failed to spawn bash: PTY or process start failed")

	// ErrSessionNotFound is returned when no session is registered or the
	// requested id does not match the active session.
	ErrSessionNotFound = errors.New("session not found")
)
