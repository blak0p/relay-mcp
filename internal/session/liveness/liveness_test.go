package liveness

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestIsAlive_OwnPID(t *testing.T) {
	t.Parallel()
	if !IsAlive(os.Getpid()) {
		t.Fatalf("IsAlive(self) = false, want true (pid=%d)", os.Getpid())
	}
}

func TestIsAlive_InitPID(t *testing.T) {
	t.Parallel()
	// In some containers PID 1 exists but is not signalable by the test
	// process (or init is absent). We only assert that IsAlive does not
	// panic and returns a bool; the semantic "init is always alive" is
	// environment-dependent.
	_ = IsAlive(1)
}

func TestIsAlive_DeadPID(t *testing.T) {
	t.Parallel()
	// A very high pid is overwhelmingly likely to be unallocated.
	dead := 99999999
	if IsAlive(dead) {
		t.Fatalf("IsAlive(%d) = true, want false", dead)
	}
}

func TestIsAlive_FlipsWhenProcessDies(t *testing.T) {
	t.Parallel()
	// Start a real child process, confirm it's alive, kill it, confirm
	// IsAlive flips to false. This triangulates against a known-alive and
	// known-dead pid of the same process.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	if !IsAlive(pid) {
		t.Fatalf("IsAlive(%d) = false right after Start, want true", pid)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	// Wait for the kernel to reap the exit so the pid is no longer alive.
	_ = cmd.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && IsAlive(pid) {
		time.Sleep(10 * time.Millisecond)
	}
	if IsAlive(pid) {
		t.Fatalf("IsAlive(%d) = true after kill+wait, want false", pid)
	}
}
