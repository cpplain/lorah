package loop

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// Run starts the infinite Claude Code CLI execution loop.
// It handles signal interrupts and retries on error.
// Run does not return under normal operation.
func Run(promptFile string, claudeFlags []string) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var stopping atomic.Bool
	go handleSignals(sigCh, &stopping, cancel, os.Exit)

	run(ctx, cancel, promptFile, claudeFlags, runClaude, &stopping, retryDelay)
	os.Exit(0)
}

// run is the testable core of Run.
// It loops until stopping is set, then returns.
func run(ctx context.Context, _ context.CancelFunc, promptFile string, flags []string,
	runFn func(context.Context, string, []string) error,
	stopping *atomic.Bool, delay time.Duration) {

	for {
		printSection("Lorah", colorBlue, "Starting loop...")

		if err := runFn(ctx, promptFile, flags); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s⏺ %sError%s\n", colorRed, colorBold, colorReset)
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			fmt.Fprintf(os.Stderr, "Retrying in %v...\n\n", delay)
			time.Sleep(delay)
		} else {
			printSection("Lorah", colorBlue, "Loop completed successfully")
		}

		if stopping.Load() {
			return
		}
	}
}

// handleSignals processes OS signals from sigCh.
// First signal sets stopping. Second signal cancels ctx and calls exitFn(0).
func handleSignals(sigCh <-chan os.Signal, stopping *atomic.Bool, cancel context.CancelFunc, exitFn func(int)) {
	for range sigCh {
		if !stopping.Swap(true) {
			// First signal: stop after current iteration
			fmt.Println()
			printSection("Lorah", colorBlue, "Received interrupt, stopping after current loop...")
		} else {
			// Second signal: cancel context and exit immediately
			fmt.Println()
			printSection("Lorah", colorBlue, "Received second interrupt, shutting down...")
			cancel()
			exitFn(0)
			return
		}
	}
}
