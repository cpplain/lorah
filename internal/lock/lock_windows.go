//go:build windows

package lock

import (
	"syscall"
	"unsafe"
)

// pidRunning returns true if the process with the given PID is still alive.
// Uses CreateToolhelp32Snapshot to enumerate all processes, which is the
// idiomatic Windows API approach for process existence checking.
func pidRunning(pid int) bool {
	// Create snapshot of all running processes
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(snapshot)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	// Iterate through process list to find matching PID
	err = syscall.Process32First(snapshot, &entry)
	if err != nil {
		return false
	}

	for {
		if int(entry.ProcessID) == pid {
			return true
		}
		err = syscall.Process32Next(snapshot, &entry)
		if err != nil {
			// No more processes
			break
		}
	}

	return false
}
