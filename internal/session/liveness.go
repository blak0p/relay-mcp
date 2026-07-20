package session

import "golang.org/x/sys/unix"

// IsAlive returns true if a process with the given pid exists. It uses
// unix.Kill(pid, 0) — the standard Unix "signal 0 ping": it checks existence
// without delivering a signal. No reaping side effects (unlike wait4).
//
// Platform: Linux and macOS. Windows is not a v1 target.
func IsAlive(pid int) bool {
	return unix.Kill(pid, 0) == nil
}
