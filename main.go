// Lorah - Simple infinite loop harness for Claude Code
//
// Usage: lorah <prompt-file> [claude-flags...]
//
// Runs Claude Code CLI in a continuous loop with formatted output.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cpplain/lorah/internal/loop"
)

// Version is set via ldflags during build. Default is "dev" for local builds.
var Version = "dev"

func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: lorah <prompt-file> [claude-flags...]

Simple infinite-loop harness for Claude Code.

Runs Claude Code CLI in a continuous loop with formatted output.
Retries automatically on error with a 5-second delay.

Arguments:
  <prompt-file>      Path to prompt file (required)
  [claude-flags...]  Flags passed directly to claude CLI

Examples:
  lorah prompt.md
  lorah prompt.md --settings .lorah/settings.json
  lorah prompt.md --model claude-opus-4-6 --max-turns 50

Flags:
  -V, --version    Print version and exit
  -h, --help       Show this help message
`)
}

// route dispatches CLI arguments to the appropriate handler and returns an exit code.
// Implements the routing rules in docs/design/loop.md §3.
func route(args []string, version string, runFn func(string, []string)) int {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return 1
	}

	switch args[0] {
	case "--version", "-version", "-V":
		fmt.Printf("lorah %s\n", version)
		return 0

	case "--help", "-help", "-h":
		printUsage(os.Stdout)
		return 0
	}

	if strings.HasPrefix(args[0], "-") {
		fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[0])
		printUsage(os.Stderr)
		return 1
	}

	runFn(args[0], args[1:])
	return 0
}

func main() {
	os.Exit(route(os.Args[1:], Version, loop.Run))
}
