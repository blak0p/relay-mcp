// Package liveness owns the cheap process-existence check used by the session
// lifecycle: IsAlive(pid) reports whether a process is still running using a
// signal-0 ping (unix.Kill(pid, 0)). It performs no reaping and has no side
// effects.
package liveness

import "golang.org/x/sys/unix"

// IsAlive returns true if a process with the given pid exists. It uses
// unix.Kill(pid, 0) — the standard Unix "signal 0 ping": it checks existence
// without delivering a signal. No reaping side effects (unlike wait4).
//
// Platform: Linux and macOS. Windows is not a v1 target.
func IsAlive(pid int) bool {
	return unix.Kill(pid, 0) == nil
}
