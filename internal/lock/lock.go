// Package lock provides a PID-based file lock to prevent concurrent harness runs.
package lock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const lockFileName = "harness.lock"

// ErrHarnessLocked is returned when another harness instance holds the lock.
var ErrHarnessLocked = errors.New("another harness instance is already running (lock: .lorah/harness.lock)")

// AcquireLock creates a PID-based lock file in harnessDir. If an existing lock
// belongs to a dead process it is removed and the lock is re-acquired.
// Returns the lock file path on success so the caller can pass it to ReleaseLock.
func AcquireLock(harnessDir string) (string, error) {
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		return "", fmt.Errorf("create .lorah dir: %w", err)
	}
	lockPath := filepath.Join(harnessDir, lockFileName)
	return acquireLock(lockPath)
}

// ReleaseLock removes the lock file. Safe to call with an empty path.
func ReleaseLock(lockPath string) {
	if lockPath != "" {
		os.Remove(lockPath)
	}
}

func acquireLock(lockPath string) (string, error) {
	pidStr := strconv.Itoa(os.Getpid())

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		if _, err := fmt.Fprint(f, pidStr); err != nil {
			f.Close()
			os.Remove(lockPath)
			return "", fmt.Errorf("write PID to lock file: %w", err)
		}
		if err := f.Close(); err != nil {
			os.Remove(lockPath)
			return "", fmt.Errorf("close lock file: %w", err)
		}
		return lockPath, nil
	}

	if !os.IsExist(err) {
		return "", fmt.Errorf("create lock file: %w", err)
	}

	// Lock file exists — check if the recorded PID is still running.
	data, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		return "", ErrHarnessLocked
	}

	existingPID, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
	if parseErr == nil && !pidRunning(existingPID) {
		// Stale lock from a crashed process.
		if removeErr := os.Remove(lockPath); removeErr != nil {
			return "", fmt.Errorf("remove stale lock: %w", removeErr)
		}
		return acquireLock(lockPath)
	}

	return "", ErrHarnessLocked
}
