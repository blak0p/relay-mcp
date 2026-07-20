// Package liveness provides cheap, side-effect-free liveness checks for
// PTY-backed processes. The current primitive is IsAlive(pid), which uses
// signal-zero (unix.Kill(pid, 0)) to ping the kernel without reaping or
// signaling the target.
//
// Platform: Linux and macOS. Windows is not a v1 target.
package liveness
