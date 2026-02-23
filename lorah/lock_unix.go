//go:build unix

package lorah

import (
	"errors"
	"os"
	"syscall"
)

// pidRunning returns true if the process with the given PID is still alive.
// It uses signal 0, which checks existence without delivering any signal.
// EPERM (permission denied) means the PID was reused by another user's process,
// so we treat it as "not running" to clear stale locks.
func pidRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	// EPERM means the process exists but is owned by another user.
	// This indicates the PID was reused and the lock is stale.
	if errors.Is(err, syscall.EPERM) {
		return false
	}
	// Any other error (e.g., ESRCH) means the process doesn't exist.
	return false
}
