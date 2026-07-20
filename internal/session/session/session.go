package session

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/blak0p/relay-mcp/internal/idgen"
	"github.com/blak0p/relay-mcp/internal/session/liveness"
)

// SessionState is the lifecycle state of a Session.
type SessionState string

const (
	// StateRunning is the state at creation. The stored status remains
	// "running" after bash dies until a session operation runs the lazy
	// liveness check (REQ-007).
	StateRunning SessionState = "running"

	// StateExited means bash exited cleanly (exit code 0) and the next
	// liveness check reconciled the stored state.
	StateExited SessionState = "exited"

	// StateError means bash died from a signal or non-zero exit and the
	// next liveness check reconciled the stored state.
	StateError SessionState = "error"
)

// Session represents one PTY-backed bash process.
type Session struct {
	ID        string
	PTY       *os.File  // master end of the PTY
	Cmd       *exec.Cmd // bash process handle
	PID       int       // cached for liveness checks
	StartedAt time.Time
	State     SessionState

	closed bool       // guards Close against double-close
	mu     sync.Mutex // guards State and closed
}

// New constructs a Session from a started (or about-to-start) command and its
// PTY master file. The id is generated via idgen.New(); the state is
// StateRunning; StartedAt is time.Now(). New does NOT start the process or
// touch the PTY — that is the caller's responsibility (creack/pty in the
// handler).
func New(cmd *exec.Cmd, pty *os.File) *Session {
	return &Session{
		ID:        idgen.New(),
		PTY:       pty,
		Cmd:       cmd,
		StartedAt: time.Now(),
		State:     StateRunning,
	}
}

// closeClosed tracks whether Close has already run. Guarded by s.mu so
// concurrent closes are safe.
//
// Close releases the PTY master file descriptor. It does NOT kill or wait on
// the bash process — process teardown is the caller's responsibility (e.g.
// creack/pty or the close_terminal handler). Close is idempotent: subsequent
// calls return nil without touching the (already-closed) FD.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.PTY != nil {
		return s.PTY.Close()
	}
	return nil
}

// ReconcileState performs the lazy liveness reconciliation mandated by the
// session lifecycle. If the stored state is StateRunning and the process is
// no longer alive, the state is flipped to:
//   - StateExited if the process exited with code 0 (clean exit), or if we
//     cannot determine the exit code (no Cmd, no ProcessState — assume clean).
//   - StateError if the process died from a signal or a non-zero exit code.
//
// If the process is still alive, or the state is already not StateRunning,
// ReconcileState is a no-op. It is safe to call concurrently and to call
// repeatedly. It does not panic on nil Cmd or nil ProcessState.
//
// ReconcileState is exported because sibling sub-packages under the session
// namespace (notably registry) need to trigger reconciliation on Get; it is
// not part of the external API of relay-mcp.
func (s *Session) ReconcileState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.State != StateRunning {
		return
	}
	if liveness.IsAlive(s.PID) {
		return
	}
	s.State = classifyExit(s.Cmd)
}

// classifyExit decides StateExited vs StateError from the available process
// state. When no information is available (nil Cmd or nil ProcessState) we
// assume a clean exit: the process is dead and we have no evidence of a
// signal/non-zero exit, so defaulting to StateExited is the least surprising
// choice.
func classifyExit(cmd *exec.Cmd) SessionState {
	if cmd == nil || cmd.ProcessState == nil {
		return StateExited
	}
	if cmd.ProcessState.Exited() && cmd.ProcessState.ExitCode() == 0 {
		return StateExited
	}
	return StateError
}
