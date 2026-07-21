package serror

import (
	"errors"
	"fmt"
)

// existingSessionError wraps ErrSessionAlreadyExists and carries the id of
// the already-active session so the MCP handler can surface it to the client
// (REQ-004). errors.Is(err, ErrSessionAlreadyExists) still holds.
type existingSessionError struct {
	id  string
	err error
}

func (e *existingSessionError) Error() string {
	return fmt.Sprintf("%v (existing id=%s)", e.err, e.id)
}

func (e *existingSessionError) Unwrap() error { return e.err }

// ExistingSessionID returns the id of the already-active session embedded in
// the error returned by Registry.Put, or "" if the error is not an
// existing-session error (or carries no id).
func ExistingSessionID(err error) string {
	var ese *existingSessionError
	if errors.As(err, &ese) {
		return ese.id
	}
	return ""
}

// NewExistingSessionError builds the wrapper carrying the existing id. It is
// exported because the registry sub-package (sibling under the session
// namespace) needs to construct this error on a duplicate Put.
func NewExistingSessionError(id string) error {
	return &existingSessionError{id: id, err: ErrSessionAlreadyExists}
}

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

	// ErrSessionNotAlive is returned when a write_terminal call targets a
	// session whose underlying bash process is dead or is missing from the
	// Registry. Its message includes the session id once attached by the
	// handler (REQ-WT-002).
	ErrSessionNotAlive = errors.New("session is not alive")

	// ErrWriteTooLarge is returned when a write_terminal call exceeds the
	// 1 MiB cap. The message includes the limit (1048576) and the actual
	// size (REQ-WT-003).
	ErrWriteTooLarge = errors.New("write exceeds maximum size")

	// ErrSessionClosed is returned when a write_terminal call races with a
	// close_terminal call and observes the closed flag set (REQ-WT-006).
	ErrSessionClosed = errors.New("session is closed")
)
