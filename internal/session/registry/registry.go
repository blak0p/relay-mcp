package registry

import (
	"sync"

	"github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

// Registry is the thread-safe single-session store. MCP over stdio serves
// exactly one client, so a single slot is sufficient; a map keyed by client
// id would add complexity with zero benefit (see design).
//
// All methods are safe for concurrent use.
type Registry struct {
	mu      sync.Mutex
	current *session.Session // nil when no active session
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Put stores s as the active session. It returns ErrSessionAlreadyExists
// (wrapped with the existing id) if a session is already registered; the
// existing session is NOT replaced. Use serror.ExistingSessionID(err) to read
// the id.
func (r *Registry) Put(s *session.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil {
		return serror.NewExistingSessionError(r.current.ID)
	}
	r.current = s
	return nil
}

// Get returns the active session after running lazy liveness reconciliation:
// if the stored state is StateRunning but the process is dead, the state is
// flipped to StateExited/StateError before returning. Returns ErrSessionNotFound
// if no session is registered.
func (r *Registry) Get() (*session.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return nil, serror.ErrSessionNotFound
	}
	r.current.ReconcileState()
	return r.current, nil
}
