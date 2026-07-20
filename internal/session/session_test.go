package session

import (
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

var sessionIDFormat = regexp.MustCompile(`^term_[0-9a-f]{16}$`)

func TestNew_PopulatesFields(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	cmd.Process = nil // not started; only struct fields matter here
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()
	fakePTY := w

	s := New(cmd, fakePTY)

	if s == nil {
		t.Fatal("New() returned nil")
	}
	if !sessionIDFormat.MatchString(s.ID) {
		t.Fatalf("s.ID = %q, want match %s", s.ID, sessionIDFormat.String())
	}
	if s.State != StateRunning {
		t.Fatalf("s.State = %q, want %q", s.State, StateRunning)
	}
	if s.StartedAt.IsZero() {
		t.Fatal("s.StartedAt is zero")
	}
	if time.Since(s.StartedAt) > 5*time.Second {
		t.Fatalf("s.StartedAt = %v, want within last 5s", s.StartedAt)
	}
	if s.PTY != fakePTY {
		t.Fatalf("s.PTY = %v, want %v", s.PTY, fakePTY)
	}
	if s.Cmd == nil {
		t.Fatal("s.Cmd = nil, want non-nil")
	}
}

func TestNew_GeneratesUniqueID(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()

	s1 := New(cmd, w)
	s2 := New(cmd, w)
	if s1.ID == s2.ID {
		t.Fatalf("two sessions share id %q, want unique", s1.ID)
	}
}

func TestSession_Close_ReleasesPTY(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()

	s := New(cmd, w)
	if err := s.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}

	// Writing to a closed FD must fail.
	if _, err := w.Write([]byte("x")); err == nil {
		t.Fatal("write to closed PTY succeeded, want error")
	}
}

func TestSession_Close_Idempotent(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()

	s := New(cmd, w)
	if err := s.Close(); err != nil {
		t.Fatalf("first Close() = %v, want nil", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close() = %v, want nil (idempotent)", err)
	}
}

func TestSession_Close_NilPTY(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	s := New(cmd, nil)
	// Closing a session with nil PTY must not panic.
	if err := s.Close(); err != nil {
		t.Fatalf("Close() on nil PTY = %v, want nil", err)
	}
}

// finishedCmd runs cmd and waits for it so that cmd.ProcessState is populated
// with the real exit code. Returns the cmd with ProcessState set.
func finishedCmd(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		// Non-zero exits produce a *ExitError; that's fine, we want it.
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("run %v: %v", cmd.Args, err)
		}
	}
	if cmd.ProcessState == nil {
		t.Fatalf("ProcessState nil for %v", cmd.Args)
	}
	return cmd
}

func TestSession_ReconcileState_CleanExitFlipsToExited(t *testing.T) {
	t.Parallel()
	cmd := finishedCmd(t, "true") // exit 0
	// Use a pid we know is dead (the finished command's pid is gone).
	s := New(cmd, nil)
	s.PID = cmd.Process.Pid // may have been reused; IsAlive should be false

	// Ensure the process is really gone before reconciling.
	waitForDead(t, s.PID)

	s.reconcileState()
	if s.State != StateExited {
		t.Fatalf("State = %q, want %q", s.State, StateExited)
	}
}

func TestSession_ReconcileState_SignalExitFlipsToError(t *testing.T) {
	t.Parallel()
	// Start a sleep, kill it with a signal → non-zero / signal exit.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Kill()
	_ = cmd.Wait() // populates ProcessState with signal exit

	s := New(cmd, nil)
	s.PID = pid
	waitForDead(t, pid)

	s.reconcileState()
	if s.State != StateError {
		t.Fatalf("State = %q, want %q (signal exit)", s.State, StateError)
	}
}

func TestSession_ReconcileState_IdempotentWhenNotRunning(t *testing.T) {
	t.Parallel()
	cmd := finishedCmd(t, "true")
	s := New(cmd, nil)
	s.PID = cmd.Process.Pid
	s.State = StateExited
	waitForDead(t, s.PID)

	s.reconcileState()
	if s.State != StateExited {
		t.Fatalf("State = %q, want %q (no flip from non-running)", s.State, StateExited)
	}
}

func TestSession_ReconcileState_AliveStaysRunning(t *testing.T) {
	t.Parallel()
	// A session whose pid is the test process itself is alive.
	cmd := exec.Command("true")
	s := New(cmd, nil)
	s.PID = os.Getpid()

	s.reconcileState()
	if s.State != StateRunning {
		t.Fatalf("State = %q, want %q (alive should stay running)", s.State, StateRunning)
	}
}

func TestSession_ReconcileState_NilCmdNoPanic(t *testing.T) {
	t.Parallel()
	s := New(nil, nil)
	s.PID = os.Getpid()
	// Must not panic even with nil Cmd.
	s.reconcileState()
	if s.State != StateRunning {
		t.Fatalf("State = %q, want %q", s.State, StateRunning)
	}
}

func waitForDead(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && IsAlive(pid) {
		time.Sleep(10 * time.Millisecond)
	}
	if IsAlive(pid) {
		t.Fatalf("pid %d still alive after 2s", pid)
	}
}
