// Package lorah provides the harness for long-running autonomous coding agents.
//
// This file (runner.go) implements the agent loop: session state management,
// phase selection, Claude CLI invocation, response streaming, and
// error recovery with exponential backoff.
package lorah

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// ErrInterrupted is returned by RunAgent when the context is cancelled by user interrupt (SIGINT/SIGTERM).
var ErrInterrupted = errors.New("interrupted")

// SessionState tracks the running state of the agent across iterations.
type SessionState struct {
	SessionNumber   int      `json:"session_number"`
	CompletedPhases []string `json:"completed_phases"`
}

// LoadSession loads session state from .lorah/session.json.
//
// If the file does not exist, a fresh default state is returned.
// If the file is corrupt, a warning is printed and a fresh state is returned.
// Completed phases that no longer exist in the config are pruned.
func LoadSession(cfg *HarnessConfig) (SessionState, error) {
	stateFile := filepath.Join(cfg.HarnessDir, "session.json")
	state := SessionState{
		SessionNumber:   0,
		CompletedPhases: []string{},
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		// File exists but can't be read — warn and reset
		fmt.Fprintf(os.Stderr, "Warning: could not read session state (%v), starting fresh\n", err)
		return state, nil
	}

	if jsonErr := json.Unmarshal(data, &state); jsonErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: corrupt session state (%v), starting fresh\n", jsonErr)
		return SessionState{SessionNumber: 0, CompletedPhases: []string{}}, nil
	}

	// Ensure CompletedPhases is never nil
	if state.CompletedPhases == nil {
		state.CompletedPhases = []string{}
	}

	// Note: With fixed phases (initialization, implementation), we don't need to prune
	// completed phases since the phase structure is unchanging.

	return state, nil
}

// SaveSession saves session state to .lorah/session.json atomically.
//
// It writes to a temp file in the same directory, then renames it to avoid
// partial writes. Errors are logged as warnings rather than propagated, to
// avoid crashing the agent loop on transient filesystem issues.
func SaveSession(cfg *HarnessConfig, state SessionState) {
	stateFile := filepath.Join(cfg.HarnessDir, "session.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal session state: %v\n", err)
		return
	}

	// Create temp file in same directory for atomic rename
	f, err := os.CreateTemp(cfg.HarnessDir, ".session.*.tmp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save session state: %v\n", err)
		return
	}
	tmpPath := f.Name()

	_, writeErr := f.Write(data)
	closeErr := f.Close()

	if writeErr != nil || closeErr != nil {
		os.Remove(tmpPath) // best-effort cleanup
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write session state: %v\n", writeErr)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to close session state temp file: %v\n", closeErr)
		}
		return
	}

	// Atomic rename
	if err := os.Rename(tmpPath, stateFile); err != nil {
		os.Remove(tmpPath) // best-effort cleanup
		fmt.Fprintf(os.Stderr, "Warning: failed to save session state: %v\n", err)
	}
}

// EvaluateCondition evaluates a phase condition string.
//
// Supported conditions:
//   - "" — always true (no condition)
//   - "exists:<path>" — true if the path exists relative to projectDir
//   - "not_exists:<path>" — true if the path does not exist
//
// Returns an error if the condition prefix is unknown or the path escapes
// the project directory (path traversal protection).
func EvaluateCondition(condition string, projectDir string) (bool, error) {
	if condition == "" {
		return true, nil
	}

	var negate bool
	var prefix string

	if strings.HasPrefix(condition, "exists:") {
		prefix = "exists:"
		negate = false
	} else if strings.HasPrefix(condition, "not_exists:") {
		prefix = "not_exists:"
		negate = true
	} else {
		return false, fmt.Errorf(
			"unknown condition prefix in %q — only 'exists:' and 'not_exists:' are supported",
			condition,
		)
	}

	relPath := condition[len(prefix):]

	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve project dir: %w", err)
	}

	target := filepath.Join(absProjectDir, relPath)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, fmt.Errorf("failed to resolve condition path: %w", err)
	}

	// Path traversal protection: target must be inside or equal to projectDir.
	// Use filepath.Rel to compute the relationship between paths.
	rel, err := filepath.Rel(absProjectDir, absTarget)
	if err != nil {
		return false, fmt.Errorf("failed to compute relative path: %w", err)
	}
	// Paths starting with ".." are outside the project directory.
	// "." (project dir itself) and subdirectories are allowed.
	if strings.HasPrefix(rel, "..") {
		return false, fmt.Errorf("path %q escapes project directory", relPath)
	}

	_, statErr := os.Stat(absTarget)
	exists := statErr == nil

	if negate {
		return !exists, nil
	}
	return exists, nil
}

// SelectPhase selects the next phase to run based on fixed phase structure.
//
// There are two fixed phases:
//   - "initialization": runs once when tracker is not initialized
//   - "implementation": loops until tracker.IsComplete() returns true
//
// Returns phase name and prompt file path.
// Returns empty strings if all work is complete.
func SelectPhase(tracker ProgressTracker, state SessionState) (string, string) {
	const initializationPhase = "initialization"
	const implementationPhase = "implementation"

	// Check if initialization phase has been completed
	initCompleted := slices.Contains(state.CompletedPhases, initializationPhase)

	// If tracker is not initialized and init hasn't run, run initialization
	if !tracker.IsInitialized() && !initCompleted {
		return initializationPhase, InitializationPromptFile
	}

	// Otherwise, run implementation phase
	return implementationPhase, ImplementationPromptFile
}

// BackoffDuration calculates the exponential backoff delay for the given
// number of consecutive errors.
//
// Formula: min(initial * multiplier^(n-1), max)
//
// Precondition: consecutiveErrors must be >= 1. The caller always increments
// consecutiveErrors before calling this function. Panics if this precondition
// is violated (indicates a programming error).
func BackoffDuration(consecutiveErrors int, cfg ErrorRecoveryConfig) time.Duration {
	if consecutiveErrors < 1 {
		panic(fmt.Sprintf("BackoffDuration: consecutiveErrors must be >= 1, got %d", consecutiveErrors))
	}
	backoff := cfg.InitialBackoffSeconds *
		math.Pow(cfg.BackoffMultiplier, float64(consecutiveErrors-1))
	if backoff > cfg.MaxBackoffSeconds {
		backoff = cfg.MaxBackoffSeconds
	}
	return time.Duration(backoff * float64(time.Second))
}

// RunAgent runs the autonomous agent loop.
//
// It:
//  1. Creates the appropriate progress tracker
//  2. Loads session state
//  3. Prints a startup banner
//  4. Loops: selects a phase, runs a Claude session, tracks state
//  5. Applies exponential backoff on errors
//  6. Prints a final summary
func RunAgent(ctx context.Context, cfg *HarnessConfig) error {
	// Ensure cursor is visible on exit (in case spinner was interrupted)
	if isTerminal(os.Stdout) {
		defer fmt.Print("\033[?25h")
	}

	// Acquire PID-based instance lock to prevent concurrent runs.
	lockPath, err := AcquireLock(cfg.HarnessDir)
	if err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer ReleaseLock(lockPath)

	// Create output manager
	om := &outputManager{writer: os.Stdout}

	// Create tracker
	tracker := NewTracker(cfg.HarnessDir)

	// Load session state
	state, err := LoadSession(cfg)
	if err != nil {
		return fmt.Errorf("failed to load session state: %w", err)
	}

	// Ensure project directory exists
	if err := os.MkdirAll(cfg.ProjectDir, 0o755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create tracking and progress files if missing (safety net)
	if err := EnsureTrackingFiles(cfg.HarnessDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// Show initial progress
	if tracker.IsInitialized() {
		if summary := tracker.Summary(); summary != "" {
			om.printLorah("%s\n", summary)
		}
	}

	// Main loop
	iteration := 0
	consecutiveErrors := 0
	lastErrorMessage := ""
	exitReason := "MAX ITERATIONS"

	for {
		iteration++
		state.SessionNumber++

		// Check for user interrupt
		if ctx.Err() != nil {
			return ErrInterrupted
		}

		// Check max iterations
		if cfg.Harness.MaxIterations != nil && iteration > *cfg.Harness.MaxIterations {
			om.printLorah("Reached max iterations (%d)\n", *cfg.Harness.MaxIterations)
			break
		}

		// Select phase
		phaseName, promptFile := SelectPhase(tracker, state)
		if phaseName == "" {
			// All work complete
			om.printLorah("All phases completed.\n")
			exitReason = "ALL COMPLETE"
			break
		}

		// Read prompt from file
		promptPath := filepath.Join(cfg.HarnessDir, promptFile)
		promptBytes, err := os.ReadFile(promptPath)
		if err != nil {
			return fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
		}
		prompt := string(promptBytes)

		// Prepend error context if previous session errored
		if lastErrorMessage != "" {
			prompt = fmt.Sprintf(
				"Note: The previous session encountered an error: %s\nPlease continue with your work.\n\n",
				truncateString(lastErrorMessage, cfg.Harness.ErrorRecovery.MaxErrorMessageLength),
			) + prompt
		}

		// Print session header
		om.printLorah("\nSESSION %d: %s\n", state.SessionNumber, strings.ToUpper(phaseName))

		// Run session (model configured in settings.json)
		result, runErr := RunSession(ctx, cfg, prompt, om)
		if runErr != nil {
			return fmt.Errorf("run session: %w", runErr)
		}

		if !result.IsError {
			consecutiveErrors = 0
			lastErrorMessage = ""

			// Mark initialization phase as completed (run_once)
			if phaseName == "initialization" && !slices.Contains(state.CompletedPhases, phaseName) {
				state.CompletedPhases = append(state.CompletedPhases, phaseName)
			}

			SaveSession(cfg, state)

			if tracker.IsComplete() {
				om.printLorah("✓ All items passing! Agent work is complete.\n")
				exitReason = "ALL COMPLETE"
				break
			}

			if summary := tracker.Summary(); summary != "" {
				om.printLorah("%s\n", summary)
			}

			om.printLorah("Agent will auto-continue in %ds...\n", cfg.Harness.AutoContinueDelay)
			time.Sleep(time.Duration(cfg.Harness.AutoContinueDelay) * time.Second)

		} else { // result.IsError == true
			errMsg := result.ErrorMsg

			SaveSession(cfg, state)

			consecutiveErrors++
			lastErrorMessage = errMsg

			backoff := BackoffDuration(consecutiveErrors, cfg.Harness.ErrorRecovery)

			om.printLorah(
				"Session encountered an error (attempt %d/%d)\n",
				consecutiveErrors,
				cfg.Harness.ErrorRecovery.MaxConsecutiveErrors,
			)

			if consecutiveErrors >= cfg.Harness.ErrorRecovery.MaxConsecutiveErrors {
				om.printLorah(
					"Reached maximum consecutive errors (%d)\n",
					cfg.Harness.ErrorRecovery.MaxConsecutiveErrors,
				)
				exitReason = "TOO MANY ERRORS"
				break
			}

			om.printLorah("Will retry with a fresh session in %.1fs...\n", backoff.Seconds())
			time.Sleep(backoff)
		}
	}

	// Final summary
	om.printLorah("%s\n", exitReason)
	if summary := tracker.Summary(); summary != "" {
		om.printLorah("%s\n", summary)
	}
	om.printLorah("\nDone!\n")
	return nil
}

// truncateString returns s truncated to maxLen characters (runes).
// Uses rune-aware truncation to avoid splitting multi-byte UTF-8 characters.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}
