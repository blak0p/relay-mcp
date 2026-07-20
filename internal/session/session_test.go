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
