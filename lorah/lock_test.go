package lorah

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestAcquireLock_CreatesLockFile verifies a lock file is created on success.
func TestAcquireLock_CreatesLockFile(t *testing.T) {
	dir := t.TempDir()

	lockPath, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock() unexpected error: %v", err)
	}
	defer ReleaseLock(lockPath)

	if lockPath == "" {
		t.Fatal("AcquireLock() returned empty lock path")
	}

	// Lock file should exist
	if _, statErr := os.Stat(lockPath); statErr != nil {
		t.Fatalf("lock file not created: %v", statErr)
	}

	// Lock file should contain the current PID
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("lock file does not contain a valid PID: %q", string(data))
	}
	if pid != os.Getpid() {
		t.Errorf("lock file PID = %d, want %d", pid, os.Getpid())
	}
}

// TestAcquireLock_CreatesHarnessDir verifies that AcquireLock creates the
// harnessDir if it does not exist.
func TestAcquireLock_CreatesHarnessDir(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "new-harness-dir")

	lockPath, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock() unexpected error: %v", err)
	}
	defer ReleaseLock(lockPath)

	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("harnessDir was not created: %v", statErr)
	}
}

// TestAcquireLock_LockedByCurrentProcess verifies that trying to acquire the
// lock while it is already held returns ErrHarnessLocked.
func TestAcquireLock_LockedByCurrentProcess(t *testing.T) {
	dir := t.TempDir()

	lockPath, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("first AcquireLock() unexpected error: %v", err)
	}
	defer ReleaseLock(lockPath)

	// Second acquire should fail because the lock is held by this process (which is alive).
	_, err = AcquireLock(dir)
	if err == nil {
		t.Fatal("second AcquireLock() expected error, got nil")
	}
	if err != ErrHarnessLocked {
		t.Errorf("second AcquireLock() error = %v, want ErrHarnessLocked", err)
	}
}

// TestAcquireLock_StaleLock verifies that a stale lock from a dead PID is
// removed and the lock is re-acquired.
func TestAcquireLock_StaleLock(t *testing.T) {
	dir := t.TempDir()

	// Write a lock file with a PID that is definitely not running.
	// PID 0 is never a valid process.
	lockFilePath := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(lockFilePath, []byte("0"), 0o644); err != nil {
		t.Fatalf("failed to write stale lock file: %v", err)
	}

	// Acquiring with a stale lock should succeed.
	lockPath, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock() with stale lock unexpected error: %v", err)
	}
	defer ReleaseLock(lockPath)

	if lockPath == "" {
		t.Fatal("AcquireLock() returned empty lock path")
	}
}

// TestReleaseLock_RemovesFile verifies that ReleaseLock removes the lock file.
func TestReleaseLock_RemovesFile(t *testing.T) {
	dir := t.TempDir()

	lockPath, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("AcquireLock() unexpected error: %v", err)
	}

	ReleaseLock(lockPath)

	if _, statErr := os.Stat(lockPath); !os.IsNotExist(statErr) {
		t.Error("lock file still exists after ReleaseLock()")
	}
}

// TestReleaseLock_EmptyPath verifies that ReleaseLock is safe to call with an
// empty string (no panic or error).
func TestReleaseLock_EmptyPath(t *testing.T) {
	// Should not panic
	ReleaseLock("")
}

// TestReleaseLock_NonexistentPath verifies that ReleaseLock is safe to call
// when the lock file no longer exists.
func TestReleaseLock_NonexistentPath(t *testing.T) {
	// Should not panic
	ReleaseLock("/tmp/does-not-exist-lock-file-xyz")
}

// TestErrHarnessLocked_ErrorString verifies the error message references
// .lorah/harness.lock.
func TestErrHarnessLocked_ErrorString(t *testing.T) {
	msg := ErrHarnessLocked.Error()
	const want = ".lorah/harness.lock"
	if !contains(msg, want) {
		t.Errorf("ErrHarnessLocked error string %q does not contain %q", msg, want)
	}
}

// TestPidRunning_CurrentProcess verifies that the current PID is reported as running.
func TestPidRunning_CurrentProcess(t *testing.T) {
	if !pidRunning(os.Getpid()) {
		t.Error("pidRunning(current PID) = false, want true")
	}
}

// TestPidRunning_InvalidPID verifies that PID 0 (never a valid process) is
// reported as not running.
func TestPidRunning_InvalidPID(t *testing.T) {
	if pidRunning(0) {
		t.Error("pidRunning(0) = true, want false")
	}
}

// contains is a helper since strings.Contains is in the strings package.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
