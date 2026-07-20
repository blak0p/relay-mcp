package session

import "sync"

// Registry is the thread-safe single-session store. MCP over stdio serves
// exactly one client, so a single slot is sufficient; a map keyed by client
// id would add complexity with zero benefit (see design).
//
// All methods are safe for concurrent use.
type Registry struct {
	mu      sync.Mutex
	current *Session // nil when no active session
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Put stores session as the active session. It returns ErrSessionAlreadyExists
// if a session is already registered; the existing session is NOT replaced.
func (r *Registry) Put(session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current != nil {
		return ErrSessionAlreadyExists
	}
	r.current = session
	return nil
}

// Get returns the active session, or ErrSessionNotFound if none is registered.
func (r *Registry) Get() (*Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil {
		return nil, ErrSessionNotFound
	}
	return r.current, nil
}