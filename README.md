# Lorah

**LO**ng-**R**unning **A**gent **H**arness

An infinite-loop harness for Claude Code CLI following the [Ralph technique](https://ghuntley.com/ralph/).

## Why Lorah?

The Ralph technique can be implemented with a simple bash loop:

```bash
while true; do cat PROMPT.md | claude -p --verbose --output-format stream-json; done
```

But you get raw `stream-json` output that's unreadable:

```
{"type":"assistant","message":{"content":[{"type":"text","text":"Let me read..."}]}}
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{"file_path":"/path/to/file"}}]}}
```

**Lorah gives you clean, color-coded output:**

```
⏺ Claude
Let me read the file

⏺ Read
/path/to/file
```

Plus automatic error recovery, graceful shutdown, and full Claude Code CLI compatibility.

**Key Features:**

- **Formatted output** - Color-coded sections and tool activity (the main reason Lorah exists)
- **Simple infinite loop** - Runs continuously until you stop it
- **Automatic error recovery** - Retries on failures with 5-second delay
- **Flag passthrough** - All Claude Code CLI flags work transparently
- **Task management** - Structured task tracking for agent workflow coordination

## Prerequisites

[Claude Code](https://claude.ai/code) - Required for agent execution

## Installation

```bash
brew install cpplain/tap/lorah
```

## Usage

Lorah is an implementation of the Ralph loop. You must understand the Ralph technique to use Lorah effectively.

Learn more about the Ralph technique: [Ralph Wiggum as a "software engineer"](https://ghuntley.com/ralph/) by Geoffrey Huntley

**Syntax:**

```bash
lorah <command> [arguments]
```

**Run loop:**

```bash
lorah run PROMPT.md
lorah run PROMPT.md --settings .lorah/settings.json
lorah run PROMPT.md --model claude-opus-4-6 --max-turns 50
```

**Task management:**

```bash
lorah task list --status=pending
lorah task create --subject="Fix auth bug" --priority=3
lorah task stats
```

## License

MIT License - see [LICENSE](LICENSE) file for details.
