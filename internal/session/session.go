// Package session models a PTY-backed bash session: the core domain object of
// relay-mcp. A Session owns the master end of a PTY and the handle to the
// spawned bash process.
package session

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/blak0p/relay-mcp/internal/idgen"
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
	PTY       *os.File      // master end of the PTY
	Cmd       *exec.Cmd     // bash process handle
	StartedAt time.Time
	State     SessionState

	mu sync.Mutex // guards State
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