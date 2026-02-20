# Lorah

A configurable harness for long-running autonomous coding agents.

## What is Lorah?

Lorah enables multi-phase agent execution by orchestrating the Claude Code CLI in isolated sessions. Instead of running Claude once and hoping for the best, Lorah breaks complex projects into phases (initialization, implementation, review) with automatic error recovery, progress tracking, and state persistence.

**Key Features:**

- **Multi-phase execution** - Break complex projects into manageable phases
- **Automatic error recovery** - Exponential backoff and retry logic
- **Progress tracking** - JSON checklists, notes files, or silent mode
- **MCP server integration** - Full tool ecosystem support
- **Session isolation** - Each phase runs in a fresh sandbox
- **State persistence** - Resume work across sessions

## Prerequisites

- [Claude Code CLI](https://claude.ai/code) - Required for agent execution
- Authentication via Claude Code login (Max/Enterprise subscription recommended) or API key

## Installation

```bash
brew install cpplain/tap/lorah
```

## Quick Start

**1. Initialize a new project**

```bash
lorah init --project-dir ./my-project
```

This scaffolds a `.lorah/` directory with starter configuration and prompts.

**2. Edit your project specification**

Open `.lorah/spec.md` and describe what you're building:

```markdown
# My Web App

## Overview

A React dashboard with real-time metrics

## Requirements

- User authentication with JWT
- Dashboard with charts
- REST API backend
```

**3. Verify your setup**

```bash
lorah verify --project-dir ./my-project
```

This checks that Claude Code CLI is accessible and your configuration is valid.

**4. Run the agent**

```bash
lorah run --project-dir ./my-project
```

Lorah executes each phase sequentially. The agent reads your spec and builds your project according to the configured phases.

## Learn More

- **[Setup Guide](docs/setup-guide.md)** - Detailed configuration and usage
- **[Examples](examples/)** - Sample projects with working configurations
  - [Simple Calculator](examples/simple-calculator/) - Basic Python CLI app
  - [Claude.ai Clone](examples/claude-ai-clone/) - Full-stack web application

## How It Works

Lorah runs a loop that executes **phases** sequentially. Each phase invokes the Claude Code CLI subprocess with a configured prompt. This design follows the patterns described in Anthropic's [Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) article.

Configuration is driven entirely by `.lorah/config.json`:

```json
{
  "model": "claude-opus-4-6",
  "permission_mode": "prompt",
  "phases": [
    {
      "name": "initialization",
      "prompt": "file:prompts/initialization.md",
      "run_once": true
    },
    {
      "name": "implementation",
      "prompt": "file:prompts/implementation.md"
    }
  ]
}
```

Phases can be `run_once` (skip after first completion) and have path-based conditions (`exists:`, `not_exists:`). Session state persists in `.lorah/session.json`.

## License

MIT License - see [LICENSE](LICENSE) file for details.
