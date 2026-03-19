package loop

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// captureOutput redirects os.Stdout and os.Stderr, runs fn, then restores them.
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

// TestRun_IteratesUntilStopping verifies the loop calls runFn on each iteration
// and exits when stopping is set after the iteration completes.
func TestRun_IteratesUntilStopping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopping atomic.Bool
	callCount := 0

	runFn := func(_ context.Context, _ string, _ []string) error {
		callCount++
		if callCount >= 3 {
			stopping.Store(true)
		}
		return nil
	}

	captureOutput(func() {
		run(ctx, cancel, "test.md", nil, runFn, &stopping, 0)
	})

	if callCount != 3 {
		t.Errorf("expected 3 iterations, got %d", callCount)
	}
}

// TestRun_PrintsStartSection verifies "Starting loop..." appears in stdout on each iteration.
func TestRun_PrintsStartSection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopping atomic.Bool
	stopping.Store(true) // exit after first iteration

	runFn := func(_ context.Context, _ string, _ []string) error {
		return nil
	}

	stdout, _ := captureOutput(func() {
		run(ctx, cancel, "test.md", nil, runFn, &stopping, 0)
	})

	if !strings.Contains(stdout, "Starting loop...") {
		t.Errorf("expected 'Starting loop...' in stdout, got: %q", stdout)
	}
}

// TestRun_PrintsSuccessSection verifies "Loop completed successfully" appears after a successful iteration.
func TestRun_PrintsSuccessSection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopping atomic.Bool
	stopping.Store(true) // exit after first iteration

	runFn := func(_ context.Context, _ string, _ []string) error {
		return nil
	}

	stdout, _ := captureOutput(func() {
		run(ctx, cancel, "test.md", nil, runFn, &stopping, 0)
	})

	if !strings.Contains(stdout, "Loop completed successfully") {
		t.Errorf("expected 'Loop completed successfully' in stdout, got: %q", stdout)
	}
}

// TestRun_ErrorHandling verifies that on runFn error, the error is printed to stderr
// with a retry message, and the loop continues.
func TestRun_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopping atomic.Bool
	callCount := 0

	runFn := func(_ context.Context, _ string, _ []string) error {
		callCount++
		stopping.Store(true) // exit after this (error) iteration
		return errors.New("some error occurred")
	}

	_, stderr := captureOutput(func() {
		run(ctx, cancel, "test.md", nil, runFn, &stopping, 0)
	})

	if !strings.Contains(stderr, "Error") {
		t.Errorf("expected 'Error' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "some error occurred") {
		t.Errorf("expected error message in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "Retrying in") {
		t.Errorf("expected 'Retrying in' in stderr, got: %q", stderr)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

// TestRun_StoppingFlagExits verifies run returns without calling runFn again
// once stopping is set.
func TestRun_StoppingFlagExits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stopping atomic.Bool
	callCount := 0

	runFn := func(_ context.Context, _ string, _ []string) error {
		callCount++
		stopping.Store(true)
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		captureOutput(func() {
			run(ctx, cancel, "test.md", nil, runFn, &stopping, 0)
		})
	}()

	select {
	case <-done:
		// expected: run returned
	case <-time.After(500 * time.Millisecond):
		t.Fatal("run did not return after stopping flag was set")
	}

	if callCount != 1 {
		t.Errorf("expected 1 iteration before stopping, got %d", callCount)
	}
}

// TestHandleSignals_FirstSignal verifies that the first signal sets stopping
// but does not cancel the context.
func TestHandleSignals_FirstSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	var stopping atomic.Bool

	// Signal that handleSignals has processed the first signal.
	processed := make(chan struct{})
	exitFn := func(int) { close(processed) }

	go func() {
		captureOutput(func() {
			handleSignals(sigCh, &stopping, cancel, exitFn)
		})
	}()

	sigCh <- os.Interrupt

	// Wait for stopping to be set (poll briefly).
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if stopping.Load() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !stopping.Load() {
		t.Error("expected stopping to be true after first signal")
	}
	if ctx.Err() != nil {
		t.Error("expected context to remain active after first signal")
	}

	// Send second signal to let the goroutine exit cleanly.
	sigCh <- os.Interrupt
	select {
	case <-processed:
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for handleSignals goroutine to exit")
	}
}

// TestHandleSignals_SecondSignal verifies that the second signal cancels the context
// and calls exitFn(0).
func TestHandleSignals_SecondSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 2)
	var stopping atomic.Bool

	var exitCode int
	var exitCalled bool
	exitFn := func(code int) {
		exitCode = code
		exitCalled = true
	}

	// Pre-queue both signals so handleSignals processes them synchronously.
	sigCh <- os.Interrupt
	sigCh <- os.Interrupt

	captureOutput(func() {
		handleSignals(sigCh, &stopping, cancel, exitFn)
	})

	if !stopping.Load() {
		t.Error("expected stopping to be true")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled after second signal")
	}
	if !exitCalled {
		t.Error("expected exitFn to be called on second signal")
	}
	if exitCode != 0 {
		t.Errorf("expected exitFn called with 0, got %d", exitCode)
	}
}
