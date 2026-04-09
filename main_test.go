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
			stdout, stderr := captureOutput(func() {
				code = route([]string{flag}, "1.2.3", nil)
			})
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stdout, "lorah 1.2.3") {
				t.Errorf("expected %q in stdout, got %q", "lorah 1.2.3", stdout)
			}
			if stderr != "" {
				t.Errorf("expected empty stderr, got %q", stderr)
			}
		})
	}
}

// TestRoute_Help verifies that --help prints usage to stdout per UNIX
// convention (an explicit documentation request is not an error).
func TestRoute_Help(t *testing.T) {
	for _, flag := range []string{"--help", "-help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(func() {
				code = route([]string{flag}, "dev", nil)
			})
			if code != 0 {
				t.Errorf("expected exit 0, got %d", code)
			}
			if !strings.Contains(stdout, "Usage:") {
				t.Errorf("expected usage in stdout, got %q", stdout)
			}
			if stderr != "" {
				t.Errorf("expected empty stderr, got %q", stderr)
			}
		})
	}
}

// TestRoute_NoArgs verifies that no arguments prints usage to stderr and exits 1.
// This is distinct from --help which exits 0 (per loop.md §3).
func TestRoute_NoArgs(t *testing.T) {
	var code int
	_, stderr := captureOutput(func() {
		code = route([]string{}, "dev", nil)
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected usage in stderr, got %q", stderr)
	}
}

// TestRoute_PromptFile verifies that a non-flag first argument is passed to
// runFn as the prompt file, with remaining args passed through as claude flags
// (loop.md §3 rule 5).
func TestRoute_PromptFile(t *testing.T) {
	var calledFile string
	var calledFlags []string
	runFn := func(file string, flags []string) {
		calledFile = file
		calledFlags = flags
	}

	code := route([]string{"prompt.md", "--max-turns", "50"}, "dev", runFn)
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

// TestRoute_UnknownFlag verifies that a leading unknown flag is rejected with
// a helpful error rather than being treated as a prompt file (loop.md §3 rule 4).
func TestRoute_UnknownFlag(t *testing.T) {
	var code int
	_, stderr := captureOutput(func() {
		code = route([]string{"--nope"}, "dev", nil)
	})
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Unknown flag: --nope") {
		t.Errorf("expected %q in stderr, got %q", "Unknown flag: --nope", stderr)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("expected usage in stderr, got %q", stderr)
	}
}
