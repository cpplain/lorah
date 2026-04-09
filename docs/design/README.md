# Lorah — Design Specifications

Design documentation for `lorah`, a simple infinite-loop harness for Claude Code.

## Foundation

Read these first. They define the harness's behavior end-to-end.

| Spec                   | Description                                                              |
| ---------------------- | ------------------------------------------------------------------------ |
| [loop.md](loop.md)     | CLI routing, loop lifecycle, signal handling, subprocess execution       |
| [output.md](output.md) | Output system: stream-JSON parsing, color-coded formatting, tool display |

## Glossary

| Term               | Definition                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------ |
| **prompt file**    | A markdown file containing instructions for the Claude Code agent, piped to `claude` stdin |
| **loop iteration** | One complete execution of the Claude Code CLI subprocess from start to finish              |
| **stream-JSON**    | Newline-delimited JSON output by Claude Code CLI's `--output-format=stream-json` flag      |
| **claude flags**   | Arguments passed through to the `claude` CLI unchanged after the prompt file               |

## Design Principles

- **Ralph Philosophy**: the agent is smart enough to manage its own workflow; the harness provides the loop and nice output, nothing more
- **Radical Simplicity**: the simplest solution that works is the best solution; prefer deleting code over adding it
- **Agent in Control**: the harness provides the loop and output; the agent reads the codebase, decides what to do, and makes progress
- **No Ceremony**: no config files, session state, lock files, or scaffolding commands
- **No External Dependencies**: stdlib only; `claude` CLI is the only runtime requirement
