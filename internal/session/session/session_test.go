package session

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"

	serror "github.com/blak0p/relay-mcp/internal/session/error"
	"github.com/blak0p/relay-mcp/internal/session/liveness"
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

	s.ReconcileState()
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

	s.ReconcileState()
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

	s.ReconcileState()
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

	s.ReconcileState()
	if s.State != StateRunning {
		t.Fatalf("State = %q, want %q (alive should stay running)", s.State, StateRunning)
	}
}

func TestSession_ReconcileState_NilCmdNoPanic(t *testing.T) {
	t.Parallel()
	s := New(nil, nil)
	s.PID = os.Getpid()
	// Must not panic even with nil Cmd.
	s.ReconcileState()
	if s.State != StateRunning {
		t.Fatalf("State = %q, want %q", s.State, StateRunning)
	}
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

// --- T-WT-03: Session.Write skeleton (size cap + closed gate) ---

// TestWrite_RejectsOversize proves the 1 MiB cap is enforced BEFORE any PTY
// write or lock acquisition. REQ-WT-003.
func TestWrite_RejectsOversize(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := New(cmd, w)
	s.PID = os.Getpid() // alive, so the liveness gate would not be the cause

	oversize := make([]byte, MaxWriteBytes+1)
	n, err := s.Write(oversize)
	if n != 0 {
		t.Fatalf("n = %d, want 0 on oversize rejection", n)
	}
	if !errors.Is(err, serror.ErrWriteTooLarge) {
		t.Fatalf("err = %v, want errors.Is(_, ErrWriteTooLarge)", err)
	}
	if !strings.Contains(err.Error(), "1048576") || !strings.Contains(err.Error(), "1048577") {
		t.Fatalf("err = %q, want message containing both 1048576 and 1048577", err.Error())
	}
}

// TestWrite_RejectsClosedSession proves the closed flag is observed under
// writeMu and returns ErrSessionClosed. REQ-WT-006 (session boundary).
func TestWrite_RejectsClosedSession(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	s := New(cmd, w)
	s.PID = os.Getpid()

	if err := s.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
	n, err := s.Write([]byte("hi"))
	if n != 0 {
		t.Fatalf("n = %d, want 0 on closed session", n)
	}
	if !errors.Is(err, serror.ErrSessionClosed) {
		t.Fatalf("err = %v, want errors.Is(_, ErrSessionClosed)", err)
	}
}

// TestWrite_AcquiresWriteMu proves writeMu is held for the duration of the
// PTY write: a second Write does not enter its critical section until the
// first's PTY write returns. We use a stub writer whose Write blocks on a
// channel, injected via the test-only setPtyWriter hook.
func TestWrite_AcquiresWriteMu(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := New(cmd, w)
	s.PID = os.Getpid()

	stub := &blockingWriter{entered: make(chan struct{}), release: make(chan struct{})}
	s.setPtyWriterForTest(stub)

	firstDone := make(chan error, 1)
	go func() {
		_, err := s.Write([]byte("first"))
		firstDone <- err
	}()

	// Wait for the first Write to enter its PTY write (blocking on the
	// stub's release channel). This proves writeMu is held.
	select {
	case <-stub.entered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("first Write never reached the PTY write; writeMu not acquired before PTY I/O")
	}

	// The second Write must block on writeMu while the first holds it.
	secondDone := make(chan error, 1)
	go func() {
		_, err := s.Write([]byte("second"))
		secondDone <- err
	}()
	select {
	case err := <-secondDone:
		t.Fatalf("second Write returned %v before first released writeMu", err)
	case <-time.After(100 * time.Millisecond):
		// expected: blocked on writeMu
	}

	// Release the first Write.
	close(stub.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Write = %v, want nil", err)
	}
	// Now the second Write proceeds and completes.
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second Write = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second Write did not complete after first released writeMu")
	}
}

// blockingWriter is a test stub that records when its Write is entered and
// blocks until the release channel is closed.
type blockingWriter struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (b *blockingWriter) Write(p []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	return len(p), nil
}

// --- T-WT-04: PTY wire + liveness ---

// TestWrite_HappyPath proves the PTY write is wired: writing "echo hi\n" to a
// real bash session delivers the bytes and returns (8, nil). REQ-WT-001.
func TestWrite_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash available; skipping PTY integration test")
	}
	cmd := exec.Command(bash, "--norc", "-i")
	ws := &pty.Winsize{Rows: 30, Cols: 100}
	ptyFile, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		t.Fatalf("pty.StartWithSize: %v", err)
	}
	t.Cleanup(func() {
		_ = ptyFile.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	s := New(cmd, ptyFile)
	s.PID = cmd.Process.Pid

	n, err := s.Write([]byte("echo hi\n"))
	if err != nil {
		t.Fatalf("Write = (%d, %v), want (8, nil)", n, err)
	}
	if n != 8 {
		t.Fatalf("n = %d, want 8 (len(\"echo hi\\n\"))", n)
	}
}

// TestWrite_DeadSessionFlipsToError proves the liveness gate: after the bash
// process dies, Write returns ErrSessionNotAlive and the session State flips
// to StateError. REQ-WT-002.
func TestWrite_DeadSessionFlipsToError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash available; skipping PTY integration test")
	}
	cmd := exec.Command(bash, "--norc", "-i")
	ws := &pty.Winsize{Rows: 30, Cols: 100}
	ptyFile, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		t.Fatalf("pty.StartWithSize: %v", err)
	}
	t.Cleanup(func() {
		_ = ptyFile.Close()
		if cmd.Process != nil {
			_, _ = cmd.Process.Wait()
		}
	})

	s := New(cmd, ptyFile)
	s.PID = cmd.Process.Pid

	// Kill the bash process and reap it so the PID is no longer alive.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill bash: %v", err)
	}
	_, _ = cmd.Process.Wait() // reap so IsAlive flips to false
	waitForDead(t, s.PID)

	n, err := s.Write([]byte("post-mortem\n"))
	if n != 0 {
		t.Fatalf("n = %d, want 0 on dead session", n)
	}
	if !errors.Is(err, serror.ErrSessionNotAlive) {
		t.Fatalf("err = %v, want errors.Is(_, ErrSessionNotAlive)", err)
	}
	if !strings.Contains(err.Error(), s.ID) {
		t.Fatalf("err = %q, want message containing session id %q", err.Error(), s.ID)
	}
	if s.State != StateError {
		t.Fatalf("s.State = %q, want %q (flipped on dead-PID detection)", s.State, StateError)
	}
}

// --- T-WT-05: partial write contract ---

// TestWrite_PartialWrite proves the partial-write contract: when the PTY
// accepts fewer bytes than requested, Write returns (N, nil) with N < len(data)
// and does NOT internally retry the remainder. REQ-WT-004.
func TestWrite_PartialWrite(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("true")
	r, w, _ := os.Pipe()
	defer r.Close()
	defer w.Close()
	s := New(cmd, w)
	s.PID = os.Getpid() // alive, so the liveness gate passes

	// stubWriter accepts only the first N bytes of any Write call and
	// records how many times Write was invoked (must be exactly 1 — no retry).
	stub := &partialStub{acceptN: 4}
	s.setPtyWriterForTest(stub)

	data := []byte("0123456789") // len 10
	n, err := s.Write(data)
	if err != nil {
		t.Fatalf("Write = (%d, %v), want (4, nil)", n, err)
	}
	if n != 4 {
		t.Fatalf("n = %d, want 4 (partial count, no retry)", n)
	}
	if stub.calls != 1 {
		t.Fatalf("stub.Write called %d time(s), want exactly 1 (no internal retry)", stub.calls)
	}
	if !bytes.Equal(stub.buf, data[:4]) {
		t.Fatalf("stub received %q, want %q (only first N bytes)", stub.buf, data[:4])
	}
}

// partialStub is a test stub that accepts only the first acceptN bytes of
// each Write call and records the call count so the test can prove no retry.
type partialStub struct {
	acceptN int
	buf     []byte
	calls   int
}

func (p *partialStub) Write(b []byte) (int, error) {
	p.calls++
	n := len(b)
	if p.acceptN > 0 && p.acceptN < n {
		n = p.acceptN
	}
	p.buf = append(p.buf, b[:n]...)
	return n, nil
}

// --- T-WT-06: concurrent writes serialization ---

// TestWrite_ConcurrentWritesSerialize proves writeMu serializes concurrent
// writes so their byte streams do not interleave in the PTY (REQ-WT-005).
// 10 goroutines each write a 100-byte payload with a distinct 2-byte prefix
// followed by 98 bytes of padding unique to that goroutine. After all
// complete, the bash output is read and we assert:
//   - the total payload bytes match 10*100
//   - each prefix P0..P9 appears exactly once
//   - no interleaving: the bytes between any two prefixes are the padding
//     of exactly one goroutine (the block is contiguous).
func TestWrite_ConcurrentWritesSerialize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no bash available; skipping PTY integration test")
	}

	// Use a non-interactive bash that reads stdin verbatim and echoes it.
	// `cat` inside bash gives us a clean byte pipe with no prompt noise.
	cmd := exec.Command(bash, "--norc", "-c", "cat")
	ws := &pty.Winsize{Rows: 30, Cols: 100}
	ptyFile, err := pty.StartWithSize(cmd, ws)
	if err != nil {
		t.Fatalf("pty.StartWithSize: %v", err)
	}
	t.Cleanup(func() {
		_ = ptyFile.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})

	s := New(cmd, ptyFile)
	s.PID = cmd.Process.Pid

	const goroutines = 10
	const payloadLen = 100

	// Build 10 distinct payloads: "P<i>" + 98 bytes of padding unique per
	// goroutine (a repeated byte distinct from all other goroutines' bytes).
	payloads := make([][]byte, goroutines)
	for i := 0; i < goroutines; i++ {
		p := make([]byte, payloadLen)
		p[0] = 'P'
		p[1] = byte('0' + i) // P0..P9
		pad := byte('a' + i) // a..j — distinct per goroutine
		for j := 2; j < payloadLen; j++ {
			p[j] = pad
		}
		payloads[i] = p
	}

	// Background reader collects everything cat echoes back. Read in a
	// goroutine until we have at least goroutines*payloadLen bytes; the main
	// goroutine fires the concurrent writes, then collects the output.
	outCh := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		tmp := make([]byte, 4096)
		for buf.Len() < goroutines*payloadLen {
			n, err := ptyFile.Read(tmp)
			if n > 0 {
				buf.Write(tmp[:n])
			}
			if err != nil {
				break
			}
		}
		outCh <- buf.Bytes()
	}()

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n, err := s.Write(payloads[i])
			if err != nil {
				t.Errorf("goroutine %d Write: %v", i, err)
				return
			}
			if n != payloadLen {
				t.Errorf("goroutine %d n = %d, want %d", i, n, payloadLen)
			}
		}(i)
	}
	wg.Wait()

	var out []byte
	select {
	case out = <-outCh:
	case <-time.After(3 * time.Second):
		_ = ptyFile.Close() // unblock the reader
		out = <-outCh
	}

	// Close the PTY so `cat` exits (cleanup will also kill it).
	_ = ptyFile.Close()

	// The PTY echoes input by default in raw mode; with `cat` the output is
	// the concatenation of the 10 payloads in some order. Extract the 10
	// blocks by scanning for "P<i>" prefixes and asserting the 98 bytes
	// that follow are the matching padding.
	if len(out) < goroutines*payloadLen {
		t.Fatalf("output len = %d, want >= %d (10*100); got:\n%q", len(out), goroutines*payloadLen, out)
	}

	// Count occurrences of each prefix and verify contiguity.
	for i := 0; i < goroutines; i++ {
		prefix := []byte{'P', byte('0' + i)}
		pad := byte('a' + i)
		count := bytes.Count(out, prefix)
		if count != 1 {
			t.Fatalf("prefix P%d appeared %d time(s), want exactly 1; got:\n%q", i, count, out)
		}
		// Find the prefix and assert the next 98 bytes are all the padding.
		idx := bytes.Index(out, prefix)
		block := out[idx : idx+payloadLen]
		for j, b := range block {
			if j < 2 {
				continue
			}
			if b != pad {
				t.Fatalf("goroutine %d block has interleaved byte at offset %d: got %q, want %q; block=%q full=%q",
					i, j, b, pad, block, out)
			}
		}
	}
}
