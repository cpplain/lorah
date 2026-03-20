package loop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// execCommandContext is the exec.CommandContext function used by runClaude.
// Overridden in tests to inject a fake "claude" subprocess.
var execCommandContext = exec.CommandContext

// runClaude executes a single Claude Code CLI session with the prompt file piped to stdin.
func runClaude(ctx context.Context, promptFile string, flags []string) error {
	file, err := os.Open(promptFile)
	if err != nil {
		return fmt.Errorf("opening prompt file: %w", err)
	}
	defer file.Close()

	args := append([]string{"-p", "--output-format", "stream-json", "--verbose"}, flags...)
	cmd := execCommandContext(ctx, "claude", args...)
	cmd.Stdin = file
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting Claude Code CLI: %w", err)
	}

	printMessages(stdout)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Claude Code CLI exited with error: %w", err)
	}
	return nil
}
