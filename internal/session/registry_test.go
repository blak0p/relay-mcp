package session

import (
	"errors"
	"os"
	"os/exec"
	"strings"
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

func TestRegistry_DuplicatePutIncludesExistingID(t *testing.T) {
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
	got := ExistingSessionID(err)
	if got != first.ID {
		t.Fatalf("ExistingSessionID(err) = %q, want %q", got, first.ID)
	}
	// And it should also be present in the error message for logging.
	if !strings.Contains(err.Error(), first.ID) {
		t.Fatalf("err.Error() = %q, want it to contain %q", err.Error(), first.ID)
	}
}

func TestExistingSessionID_OnPlainError(t *testing.T) {
	t.Parallel()
	if got := ExistingSessionID(errors.New("plain error")); got != "" {
		t.Fatalf("ExistingSessionID on plain error = %q, want empty", got)
	}
}

func TestRegistry_Get_ReconcilesDeadProcessToExited(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	// finishedCmd gives us a cmd with ProcessState populated (exit 0).
	cmd := finishedCmd(t, "true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := New(cmd, w)
	s.PID = cmd.Process.Pid
	waitForDead(t, s.PID)

	if err := reg.Put(s); err != nil {
		t.Fatalf("Put = %v, want nil", err)
	}

	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if got.State != StateExited {
		t.Fatalf("after Get, State = %q, want %q (liveness must reconcile)", got.State, StateExited)
	}
}

func TestRegistry_Get_LeavesAliveSessionRunning(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := New(cmd, w)
	s.PID = os.Getpid() // alive
	if err := reg.Put(s); err != nil {
		t.Fatalf("Put = %v, want nil", err)
	}
	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if got.State != StateRunning {
		t.Fatalf("alive session State = %q, want %q", got.State, StateRunning)
	}
}
