package session

import (
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
)

func newTestSession(t *testing.T) *Session {
	t.Helper()
	cmd := exec.Command("true")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	t.Cleanup(func() { w.Close() })
	return New(cmd, w)
}

func TestRegistry_PutThenGet(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	s := newTestSession(t)

	if err := reg.Put(s); err != nil {
		t.Fatalf("Put = %v, want nil", err)
	}
	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if got != s {
		t.Fatalf("Get returned %p, want %p", got, s)
	}
}

func TestRegistry_GetEmptyReturnsErrSessionNotFound(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	_, err := reg.Get()
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Get on empty registry = %v, want ErrSessionNotFound", err)
	}
}

func TestRegistry_PutDuplicateReturnsErrSessionAlreadyExists(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	first := newTestSession(t)
	second := newTestSession(t)

	if err := reg.Put(first); err != nil {
		t.Fatalf("first Put = %v, want nil", err)
	}
	err := reg.Put(second)
	if !errors.Is(err, ErrSessionAlreadyExists) {
		t.Fatalf("second Put = %v, want ErrSessionAlreadyExists", err)
	}
	// Verify the registry still holds the first session, not the second.
	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get after duplicate Put = %v, want nil", err)
	}
	if got != first {
		t.Fatal("registry was mutated by the rejected Put")
	}
}

func TestRegistry_ConcurrentPutIsSafe(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- reg.Put(newTestSession(t))
		}()
	}
	wg.Wait()
	close(errs)

	success := 0
	for err := range errs {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrSessionAlreadyExists) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("exactly one Put must succeed, got %d", success)
	}
}