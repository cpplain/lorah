// Lorah - Simple infinite loop harness for Claude Code
//
// Usage: lorah PROMPT.md [claude-flags...]
//
// Runs an infinite loop that executes Claude CLI with stream-JSON output,
// parses messages, formats output with colors, and retries on errors.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Version is set via ldflags during build. Default is "dev" for local builds.
var Version = "dev"

const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
	colorBold  = "\033[1m"
	colorRed   = "\033[31m"

	maxBufferSize = 1024 * 1024 // 1MB buffer for JSON parsing
	retryDelay    = 5 * time.Second
)

// printSection outputs a labeled section with optional content.
func printSection(label, color, content string) {
	fmt.Printf("%s==>%s %s%s%s\n", color, colorReset, colorBold, label, colorReset)
	if content != "" {
		fmt.Printf("%s\n", content)
	}
}

func main() {
	// Handle --version / -version / -V
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version" || os.Args[1] == "-V") {
		fmt.Printf("lorah %s\n", Version)
		os.Exit(0)
	}

	// Handle --help / -help / -h or no args
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-help" || os.Args[1] == "-h" {
		fmt.Fprint(os.Stderr, `Usage: lorah <prompt-file> [claude-flags...]

Simple infinite-loop harness for Claude Code.
Runs Claude CLI in a continuous loop with formatted output.

Arguments:
  <prompt-file>      Path to prompt file (required, first argument)
  [claude-flags...]  Flags passed directly to claude CLI (optional)

Examples:
  lorah prompt.md
  lorah task.txt --settings .lorah/settings.json
  lorah instructions.md --model opus --max-turns 50

Flags:
  -V, --version      Print version and exit
  -h, --help         Show this help message
`)
		os.Exit(1)
	}

	promptFile := os.Args[1]
	claudeFlags := os.Args[2:]

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println()
		printSection("Lorah", colorBlue, "Received interrupt, shutting down...")
		cancel()
		os.Exit(0)
	}()

	// Infinite loop
	iteration := 0
	for {
		iteration++
		printSection("Lorah", colorBlue, "Starting loop...")

		if err := runClaude(ctx, promptFile, claudeFlags); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s==> %sError%s\n", colorRed, colorBold, colorReset)
			fmt.Fprintf(os.Stderr, "%v\n\n", err)
			fmt.Fprintf(os.Stderr, "Retrying in %v...\n\n", retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		// Success - continue immediately to next iteration
		printSection("Lorah", colorBlue, "Loop completed successfully")
	}
}

// runClaude executes a single Claude CLI session with the prompt file piped to stdin.
func runClaude(ctx context.Context, promptFile string, flags []string) error {
	// Open prompt file to pipe as stdin
	file, err := os.Open(promptFile)
	if err != nil {
		return fmt.Errorf("opening prompt file: %w", err)
	}
	defer file.Close()

	// Build claude command
	args := []string{
		"-p",
		"--output-format", "stream-json",
		"--verbose",
	}
	args = append(args, flags...) // Add user-provided flags

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = file // Pipe prompt file contents to stdin
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting claude CLI: %w", err)
	}

	// Print formatted output in real-time
	printMessages(stdout)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude CLI exited with error: %w", err)
	}

	return nil
}

// printMessages reads stream-JSON from r and prints formatted output.
func printMessages(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4096), maxBufferSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Skip malformed JSON silently for forward compatibility
			continue
		}

		msgType, _ := msg["type"].(string)

		switch msgType {
		case "assistant":
			msgData, ok := msg["message"].(map[string]any)
			if !ok {
				continue
			}

			contentArray, ok := msgData["content"].([]any)
			if !ok {
				continue
			}

			for _, item := range contentArray {
				block, ok := item.(map[string]any)
				if !ok {
					continue
				}

				blockType, _ := block["type"].(string)

				switch blockType {
				case "text":
					text, _ := block["text"].(string)
					printSection("Claude", colorBlue, text)

				case "thinking":
					thinking, _ := block["thinking"].(string)
					printSection("Claude (thinking)", colorBlue, thinking)

				case "tool_use":
					name, _ := block["name"].(string)
					if name == "" {
						continue
					}

					// Format tool name to title case
					toolName := strings.ToUpper(name[:1]) + strings.ToLower(name[1:])

					// Extract relevant input parameter for display
					var content string
					if input, ok := block["input"].(map[string]any); ok {
						switch name {
						case "Bash":
							content, _ = input["command"].(string)
						case "Read", "Edit", "Write":
							content, _ = input["file_path"].(string)
						case "Grep", "Glob":
							content, _ = input["pattern"].(string)
						case "WebFetch":
							content, _ = input["url"].(string)
						case "Task":
							content, _ = input["description"].(string)
						case "Agent":
							content, _ = input["prompt"].(string)
						}
					}

					// Truncate to 3 lines if longer
					if content != "" {
						lines := strings.Split(content, "\n")
						if len(lines) > 3 {
							content = strings.Join(lines[:3], "\n") + "\n..."
						}
					}

					printSection(toolName, colorGreen, content)
				}
			}

		case "result":
			isError, _ := msg["is_error"].(bool)
			if isError {
				result, _ := msg["result"].(string)
				fmt.Println()
				printSection("Result (error)", colorRed, result)
			}
		}
	}
}
