package session

import (
	"bytes"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestIntegration_BashSpawnSpawnsRunningPTY verifies REQ-001 and SCENARIO-005:
// spawning a real bash inside a PTY at 100x30, writing a command, and reading
// back output. Skipped when /bin/bash is unavailable (CI portability).
func TestIntegration_BashSpawnSpawnsRunningPTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("no /bin/bash available; skipping PTY integration test")
	}

	// Use a non-interactive bash with an explicit prompt disabled so the
	// PTY output is just our command's output. We send "echo <marker>; exit"
	// so the process terminates and the read goroutine gets EOF.
	cmd := exec.Command(bash, "--norc", "-i")
	// Wire the std streams to the PTY so bash reads/writes through it.
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

	// The spawned process must be alive immediately.
	if !IsAlive(cmd.Process.Pid) {
		t.Fatal("spawned bash is not alive immediately after StartWithSize")
	}

	// PTY window size must be exactly 100x30.
	got, err := pty.GetsizeFull(ptyFile)
	if err != nil {
		t.Fatalf("GetsizeFull: %v", err)
	}
	if got.Cols != 100 || got.Rows != 30 {
		t.Fatalf("PTY size = %dx%d, want 100x30", got.Cols, got.Rows)
	}

	// Drive bash: send "echo <marker>" + newline, read back output. The
	// command string itself is echoed once (PTY echo), and bash executing it
	// prints the marker a second time. So the marker must appear at least
	// TWICE — proving bash actually ran inside the PTY, not just that the
	// input was echoed back.
	//
	// PTY fds do not honor SetReadDeadline on all kernels, so we read in a
	// goroutine and select on a timeout.
	const marker = "relay_integration_marker"
	// Send the command with CR — the PTY line discipline maps CR to a
	// newline for the canonical-mode read. We also disable bracketed paste
	// by sending a small preamble first; some interactive bashes enable it
	// and swallow pasted input.
	if _, err := io.WriteString(ptyFile, "bind 'set enable-bracketed-paste off'\r"); err != nil {
		t.Fatalf("write preamble: %v", err)
	}
	// Drain the preamble output before sending the real command.
	time.Sleep(100 * time.Millisecond)
	if _, err := io.WriteString(ptyFile, "echo "+marker+"\r"); err != nil {
		t.Fatalf("write to PTY: %v", err)
	}

	type readChunk struct{ data []byte }
	chunks := make(chan readChunk, 64)
	readErr := make(chan error, 1)
	go func() {
		for {
			tmp := make([]byte, 4096)
			n, err := ptyFile.Read(tmp)
			if n > 0 {
				select {
				case chunks <- readChunk{data: append([]byte(nil), tmp[:n]...)}:
				default:
				}
			}
			if err != nil {
				readErr <- err
				return
			}
		}
	}()

	var buf bytes.Buffer
	deadline := time.After(5 * time.Second)
loop:
	for {
		select {
		case c := <-chunks:
			buf.Write(c.data)
			if strings.Count(buf.String(), marker) >= 2 {
				break loop
			}
		case <-deadline:
			break loop
		case err := <-readErr:
			t.Fatalf("read from PTY ended early: %v", err)
		}
	}

	out := buf.String()
	if count := strings.Count(out, marker); count < 2 {
		t.Fatalf("PTY output contained marker %d time(s), want >= 2 (input echo + echo output); got:\n%s", count, out)
	}
}
