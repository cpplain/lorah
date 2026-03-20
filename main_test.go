package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput redirects os.Stdout and os.Stderr, runs fn, then restores them.
// Returns captured stdout and stderr as strings.
func captureOutput(fn func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	fn()

	wOut.Close()
	wErr.Close()

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return string(outBytes), string(errBytes)
}

func TestRoute_Version(t *testing.T) {
	for _, flag := range []string{"--version", "-version", "-V"} {
		t.Run(flag, func(t *testing.T) {
			var code int
			stdout, _ := captureOutput(func() {
				code = route([]string{flag}, "1.2.3", nil, nil)
			})
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stdout, "lorah 1.2.3") {
				t.Errorf("expected %q in stdout, got %q", "lorah 1.2.3", stdout)
			}
		})
	}
}

func TestRoute_Help(t *testing.T) {
	for _, flag := range []string{"--help", "-help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var code int
			_, stderr := captureOutput(func() {
				code = route([]string{flag}, "dev", nil, nil)
			})
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stderr, "Usage:") {
				t.Errorf("expected usage in stderr, got %q", stderr)
			}
		})
	}
}

// TestRoute_NoArgs verifies that no arguments prints usage to stderr and exits 1.
// This is distinct from --help which exits 0 (per cli.md §3).
func TestRoute_NoArgs(t *testing.T) {
	var code int
	_, stderr := captureOutput(func() {
		code = route([]string{}, "dev", nil, nil)
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected usage in stderr, got %q", stderr)
	}
}

func TestRoute_Run(t *testing.T) {
	var calledFile string
	var calledFlags []string
	runFn := func(file string, flags []string) error {
		calledFile = file
		calledFlags = flags
		return nil
	}

	code := route([]string{"run", "prompt.md", "--max-turns", "50"}, "dev", runFn, nil)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if calledFile != "prompt.md" {
		t.Errorf("expected prompt file %q, got %q", "prompt.md", calledFile)
	}
	if len(calledFlags) != 2 || calledFlags[0] != "--max-turns" || calledFlags[1] != "50" {
		t.Errorf("unexpected flags: %v", calledFlags)
	}
}

func TestRoute_Task(t *testing.T) {
	var calledArgs []string
	taskFn := func(args []string) error {
		calledArgs = args
		return nil
	}

	code := route([]string{"task", "list"}, "dev", nil, taskFn)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	if len(calledArgs) != 1 || calledArgs[0] != "list" {
		t.Errorf("unexpected args: %v", calledArgs)
	}
}

func TestRoute_UnknownCommand(t *testing.T) {
	var code int
	_, stderr := captureOutput(func() {
		code = route([]string{"unknown"}, "dev", nil, nil)
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Unknown command: unknown") {
		t.Errorf("expected error message in stderr, got %q", stderr)
	}
}
