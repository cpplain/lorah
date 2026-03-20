// Lorah - Simple infinite loop harness for Claude Code
//
// Usage: lorah <command> [arguments]
//
// Runs Claude Code CLI in a continuous loop with formatted output.
package main

import (
	"fmt"
	"os"

	"github.com/cpplain/lorah/internal/loop"
	"github.com/cpplain/lorah/internal/task"
)

// Version is set via ldflags during build. Default is "dev" for local builds.
var Version = "dev"

func printUsage() {
	fmt.Fprint(os.Stderr, `Usage: lorah <command> [arguments]

Simple infinite-loop harness for Claude Code.

Commands:
  run     Run Claude Code CLI in an infinite loop
  task    Manage tasks

Flags:
  -V, --version    Print version and exit
  -h, --help       Show this help message

Run 'lorah <command> --help' for command-specific help.
`)
}

func printRunUsage() {
	fmt.Fprint(os.Stderr, `Usage: lorah run <prompt-file> [claude-flags...]

Run Claude Code CLI in a continuous loop with formatted output.
Retries automatically on error with a 5-second delay.

Arguments:
  <prompt-file>      Path to prompt file (required)
  [claude-flags...]  Flags passed directly to claude CLI

Examples:
  lorah run prompt.md
  lorah run task.txt --settings .lorah/settings.json
  lorah run instructions.md --model claude-opus-4-6 --max-turns 50

Flags:
  -h, --help    Show this help message
`)
}

func printTaskUsage() {
	fmt.Fprint(os.Stderr, `Usage: lorah task <subcommand> [args...] [flags...]

Manage tasks stored in tasks.json.

Subcommands:
  list        List tasks
  get         Get task details
  create      Create a new task
  update      Update a task
  delete      Delete a task
  export      Export tasks to markdown

Flags:
  -h, --help    Show this help message

Run 'lorah task <subcommand> --help' for subcommand-specific help.
`)
}

// route dispatches CLI arguments to the appropriate handler and returns an exit code.
func route(args []string, version string, runFn func(string, []string) error, taskFn func([]string) error) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	switch args[0] {
	case "--version", "-version", "-V":
		fmt.Printf("lorah %s\n", version)
		return 0

	case "--help", "-help", "-h":
		printUsage()
		return 0

	case "run":
		runArgs := args[1:]
		if len(runArgs) == 0 {
			printRunUsage()
			return 1
		}
		if runArgs[0] == "--help" || runArgs[0] == "-help" || runArgs[0] == "-h" {
			printRunUsage()
			return 0
		}
		promptFile := runArgs[0]
		claudeFlags := runArgs[1:]
		if err := runFn(promptFile, claudeFlags); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		return 0

	case "task":
		taskArgs := args[1:]
		if len(taskArgs) == 0 {
			printTaskUsage()
			return 1
		}
		if taskArgs[0] == "--help" || taskArgs[0] == "-help" || taskArgs[0] == "-h" {
			printTaskUsage()
			return 0
		}
		if err := taskFn(taskArgs); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		printUsage()
		return 1
	}
}

func runCmd(promptFile string, claudeFlags []string) error {
	loop.Run(promptFile, claudeFlags)
	return nil
}

func taskCmd(args []string) error {
	storage := task.NewJSONStorage("tasks.json")
	code := task.HandleTask(args, os.Stdout, storage)
	if code != 0 {
		return fmt.Errorf("task command failed")
	}
	return nil
}

func main() {
	os.Exit(route(os.Args[1:], Version, runCmd, taskCmd))
}
