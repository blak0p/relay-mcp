package registry

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/liveness"
	"github.com/blak0p/relay-mcp/internal/session/session"
)

func newTestSession(t *testing.T) *session.Session {
	t.Helper()
	cmd := exec.Command("true")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	t.Cleanup(func() { w.Close() })
	return session.New(cmd, w)
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
	if !errors.Is(err, serror.ErrSessionNotFound) {
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
	if !errors.Is(err, serror.ErrSessionAlreadyExists) {
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
		} else if !errors.Is(err, serror.ErrSessionAlreadyExists) {
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
	if !errors.Is(err, serror.ErrSessionAlreadyExists) {
		t.Fatalf("second Put = %v, want ErrSessionAlreadyExists", err)
	}
	got := serror.ExistingSessionID(err)
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
	if got := serror.ExistingSessionID(errors.New("plain error")); got != "" {
		t.Fatalf("ExistingSessionID on plain error = %q, want empty", got)
	}
}

// finishedCmd runs cmd and waits for it so that cmd.ProcessState is populated
// with the real exit code. Returns the cmd with ProcessState set.
func finishedCmd(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("run %v: %v", cmd.Args, err)
		}
	}
	if cmd.ProcessState == nil {
		t.Fatalf("ProcessState nil for %v", cmd.Args)
	}
	return cmd
}

func waitForDead(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && liveness.IsAlive(pid) {
		time.Sleep(10 * time.Millisecond)
	}
	if liveness.IsAlive(pid) {
		t.Fatalf("pid %d still alive after 2s", pid)
	}
}

func TestRegistry_Get_ReconcilesDeadProcessToExited(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	cmd := finishedCmd(t, "true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := session.New(cmd, w)
	s.PID = cmd.Process.Pid
	waitForDead(t, s.PID)

	if err := reg.Put(s); err != nil {
		t.Fatalf("Put = %v, want nil", err)
	}

	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if got.State != session.StateExited {
		t.Fatalf("after Get, State = %q, want %q (liveness must reconcile)", got.State, session.StateExited)
	}
}

func TestRegistry_Get_LeavesAliveSessionRunning(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := session.New(cmd, w)
	s.PID = os.Getpid() // alive
	if err := reg.Put(s); err != nil {
		t.Fatalf("Put = %v, want nil", err)
	}
	got, err := reg.Get()
	if err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if got.State != session.StateRunning {
		t.Fatalf("alive session State = %q, want %q", got.State, session.StateRunning)
	}
}

func TestRegistryRelease_AbsentMismatchedAndRepeatedTargetsAreNoOps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		setup  func(t *testing.T, reg *Registry) string
		wantID string
	}{
		{
			name: "empty registry",
			setup: func(_ *testing.T, _ *Registry) string {
				return "missing"
			},
			wantID: "missing",
		},
		{
			name: "mismatched session",
			setup: func(t *testing.T, reg *Registry) string {
				s := newTestSession(t)
				if err := reg.Put(s); err != nil {
					t.Fatalf("Put: %v", err)
				}
				return "other"
			},
			wantID: "other",
		},
		{
			name: "repeated release",
			setup: func(t *testing.T, reg *Registry) string {
				s := newTestSession(t)
				if err := reg.Put(s); err != nil {
					t.Fatalf("Put: %v", err)
				}
				if _, matched, err := reg.Release(s.ID, func(*session.Session) (session.CloseResult, error) {
					return session.CloseResult{}, nil
				}); err != nil || !matched {
					t.Fatalf("initial Release() = matched %t, err %v; want matched true, nil", matched, err)
				}
				return s.ID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			id := tt.setup(t, reg)
			called := false
			_, matched, err := reg.Release(id, func(*session.Session) (session.CloseResult, error) {
				called = true
				return session.CloseResult{}, nil
			})
			if err != nil {
				t.Fatalf("Release() error = %v, want nil", err)
			}
			if matched {
				t.Fatal("Release() matched = true, want false")
			}
			if called {
				t.Fatal("Release() invoked cleanup for an absent or mismatched target")
			}
		})
	}
}

func TestRegistryRelease_BlocksPutUntilCleanupCompletesAndAlwaysClearsSlot(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	current := newTestSession(t)
	if err := reg.Put(current); err != nil {
		t.Fatalf("Put current: %v", err)
	}

	cleanupStarted := make(chan struct{})
	allowCleanup := make(chan struct{})
	releaseDone := make(chan error, 1)
	go func() {
		_, _, err := reg.Release(current.ID, func(*session.Session) (session.CloseResult, error) {
			close(cleanupStarted)
			<-allowCleanup
			return session.CloseResult{}, errors.New("injected cleanup failure")
		})
		releaseDone <- err
	}()
	<-cleanupStarted

	putDone := make(chan error, 1)
	go func() { putDone <- reg.Put(newTestSession(t)) }()
	select {
	case err := <-putDone:
		t.Fatalf("Put returned %v before cleanup completed", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(allowCleanup)
	if err := <-releaseDone; err == nil {
		t.Fatal("Release() error = nil, want injected cleanup failure")
	}
	if err := <-putDone; err != nil {
		t.Fatalf("Put after cleanup = %v, want nil", err)
	}
}

func TestRegistryRelease_ClosesExitedAndErroredSessions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		script    string
		wantState session.SessionState
		wantExit  int
	}{
		{name: "clean exit", script: "exit 0", wantState: session.StateExited, wantExit: 0},
		{name: "failed exit", script: "exit 7", wantState: session.StateError, wantExit: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			cmd := exec.Command("bash", "-c", tt.script)
			if err := cmd.Run(); err != nil {
				if _, ok := err.(*exec.ExitError); !ok {
					t.Fatalf("cmd.Run: %v", err)
				}
			}
			s := session.New(cmd, nil)
			s.PID = cmd.Process.Pid
			if err := reg.Put(s); err != nil {
				t.Fatalf("Put: %v", err)
			}

			result, matched, err := reg.Release(s.ID, func(s *session.Session) (session.CloseResult, error) {
				return s.Shutdown(time.Second)
			})
			if err != nil {
				t.Fatalf("Release() error = %v, want nil", err)
			}
			if !matched {
				t.Fatal("Release() matched = false, want true")
			}
			if result.State != tt.wantState || result.ExitCode != tt.wantExit {
				t.Fatalf("Release() result = %+v, want state %q exit %d", result, tt.wantState, tt.wantExit)
			}
			if err := reg.Put(newTestSession(t)); err != nil {
				t.Fatalf("Put after Release = %v, want nil", err)
			}
		})
	}
}
