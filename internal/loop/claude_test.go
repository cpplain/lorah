package loop

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperProcess is not a real test — it's a helper that acts as the fake
// "claude" CLI process when invoked by tests using the TestHelperProcess pattern.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}

	// Find args after the "--" separator (everything before is test framework args).
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}

	// Write the received args to a file if the test requested it.
	if f := os.Getenv("GO_TEST_ARGS_FILE"); f != "" {
		_ = os.WriteFile(f, []byte(strings.Join(args, "\n")), 0600)
	}

	// Write stdin content to a file if the test requested it.
	if f := os.Getenv("GO_TEST_STDIN_FILE"); f != "" {
		data, _ := io.ReadAll(os.Stdin)
		_ = os.WriteFile(f, data, 0600)
	} else {
		_, _ = io.Copy(io.Discard, os.Stdin)
	}

	// Exit with the specified code (default 0).
	if os.Getenv("GO_TEST_EXIT_CODE") == "1" {
		os.Exit(1)
	}
	os.Exit(0)
}

// helperClaudeFunc returns an execCommandContext override that runs the test
// binary itself as the "claude" subprocess via TestHelperProcess.
func helperClaudeFunc(extraEnv ...string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, arg ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--"}, arg...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		env := append(os.Environ(), "GO_TEST_HELPER_PROCESS=1")
		cmd.Env = append(env, extraEnv...)
		return cmd
	}
}

// TestRunClaude_CorrectArgs verifies that runClaude passes the expected args to claude:
// -p, --output-format, stream-json, --verbose, followed by any passthrough flags.
func TestRunClaude_CorrectArgs(t *testing.T) {
	argsFile := t.TempDir() + "/args.txt"

	old := execCommandContext
	defer func() { execCommandContext = old }()
	execCommandContext = helperClaudeFunc("GO_TEST_ARGS_FILE=" + argsFile)

	promptFile := t.TempDir() + "/prompt.md"
	if err := os.WriteFile(promptFile, []byte("test prompt"), 0600); err != nil {
		t.Fatal(err)
	}

	captureOutput(func() {
		_ = runClaude(context.Background(), promptFile, []string{"--extra-flag"})
	})

	argsData, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args file not written (runClaude never called execCommandContext): %v", err)
	}
	got := string(argsData)

	for _, want := range []string{"-p", "--output-format", "stream-json", "--verbose", "--extra-flag"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected arg %q in claude args, got: %q", want, got)
		}
	}
}

// TestRunClaude_PromptFilePipedToStdin verifies that the prompt file contents are
// piped to the claude subprocess stdin.
func TestRunClaude_PromptFilePipedToStdin(t *testing.T) {
	stdinFile := t.TempDir() + "/stdin.txt"

	old := execCommandContext
	defer func() { execCommandContext = old }()
	execCommandContext = helperClaudeFunc("GO_TEST_STDIN_FILE=" + stdinFile)

	promptContent := "this is the prompt content"
	promptFile := t.TempDir() + "/prompt.md"
	if err := os.WriteFile(promptFile, []byte(promptContent), 0600); err != nil {
		t.Fatal(err)
	}

	captureOutput(func() {
		_ = runClaude(context.Background(), promptFile, nil)
	})

	stdinData, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("stdin file not written (runClaude never piped stdin): %v", err)
	}
	if string(stdinData) != promptContent {
		t.Errorf("expected stdin %q, got %q", promptContent, string(stdinData))
	}
}

// TestRunClaude_MissingPromptFile verifies that runClaude returns an error with the
// "opening prompt file:" prefix when the prompt file does not exist.
func TestRunClaude_MissingPromptFile(t *testing.T) {
	old := execCommandContext
	defer func() { execCommandContext = old }()
	execCommandContext = helperClaudeFunc()

	err := runClaude(context.Background(), "/nonexistent/path/to/prompt.md", nil)
	if err == nil {
		t.Fatal("expected error for missing prompt file, got nil")
	}
	if !strings.HasPrefix(err.Error(), "opening prompt file:") {
		t.Errorf("expected error prefix %q, got: %q", "opening prompt file:", err.Error())
	}
}

// TestRunClaude_SubprocessError verifies that a non-zero exit from claude returns
// an error prefixed "Claude Code CLI exited with error:".
func TestRunClaude_SubprocessError(t *testing.T) {
	old := execCommandContext
	defer func() { execCommandContext = old }()
	execCommandContext = helperClaudeFunc("GO_TEST_EXIT_CODE=1")

	promptFile := t.TempDir() + "/prompt.md"
	if err := os.WriteFile(promptFile, []byte("prompt"), 0600); err != nil {
		t.Fatal(err)
	}

	err := runClaude(context.Background(), promptFile, nil)
	if err == nil {
		t.Fatal("expected error for failing subprocess, got nil")
	}
	if !strings.HasPrefix(err.Error(), "Claude Code CLI exited with error:") {
		t.Errorf("expected error prefix %q, got: %q", "Claude Code CLI exited with error:", err.Error())
	}
}
