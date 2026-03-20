# Output Specification

---

## 1. Overview

### Purpose

The output system parses Claude Code CLI's stream-JSON stdout and displays color-coded,
human-readable sections in real-time as each line arrives.

### Goals

- **Real-time**: output displayed as stream-JSON lines arrive, not buffered to end
- **Color-coded**: section headers colored by source for visual scanning
- **Forward-compatible**: unknown message types and block types silently skipped
- **Truncated**: multi-line tool inputs condensed to first line plus remaining count

### Non-Goals

- Machine-readable (JSON) output from Lorah itself
- Log files or output persistence
- Configurable color themes or color disabling

---

## 2. Color Scheme

| Constant     | ANSI Code  | Use                  |
| ------------ | ---------- | -------------------- |
| `colorReset` | `\033[0m`  | Reset all formatting |
| `colorGreen` | `\033[32m` | Tool section icon    |
| `colorBlue`  | `\033[34m` | Lorah section icon   |
| `colorBold`  | `\033[1m`  | Section label text   |
| `colorRed`   | `\033[31m` | Error section icon   |

---

## 3. Section Format

### Signature

```go
func printSection(label, color, content string)
```

### Output Template

```
<color>⏺<reset> <bold><label><reset>
<content trimmed>

```

- The icon (`⏺`) is colored; the label is bold; both are reset after
- If `content` is empty, only the icon+label line and blank line are printed
- Content is trimmed of leading and trailing whitespace before printing

---

## 4. Stream-JSON Parsing

### Signature

```go
func printMessages(r io.Reader)
```

### Scanner Configuration

- `bufio.Scanner` reading from `r` line-by-line
- Buffer initialized to 4096 bytes, max `maxBufferSize` (1MB)
- Empty lines are skipped
- Lines that fail JSON unmarshal are silently skipped (forward compatibility)

### Top-Level Message Types

| `msg["type"]`   | Handling                                                     |
| --------------- | ------------------------------------------------------------ |
| `"assistant"`   | Parse `msg["message"]["content"]` as array of content blocks |
| `"result"`      | Display only if `msg["is_error"]` is `true`                  |
| (anything else) | Silently skipped                                             |

### Content Block Types (within `assistant` messages)

| `block["type"]` | Display                                                    |
| --------------- | ---------------------------------------------------------- |
| `"text"`        | `printSection("Claude", "", block["text"])`                |
| `"thinking"`    | `printSection("Claude (thinking)", "", block["thinking"])` |
| `"tool_use"`    | Tool display (see section 5)                               |
| (anything else) | Silently skipped                                           |

### Result Messages

Only error results are displayed:

```
msg["is_error"] == true  →  printSection("Result (error)", colorRed, msg["result"])
```

Non-error result messages are silently skipped.

---

## 5. Tool Display

### Tool Name Formatting

The raw tool name from stream-JSON is title-cased for display:

```go
toolName := strings.ToUpper(name[:1]) + name[1:]
```

Example: `"tool_use"` name `"Bash"` → displayed as `"Bash"`.

### Tool Input Parameter Extraction

One input parameter is extracted per tool for the section content:

| Tool Name  | Input Key     | What Is Shown                |
| ---------- | ------------- | ---------------------------- |
| `Bash`     | `command`     | Shell command being executed |
| `Read`     | `file_path`   | File being read              |
| `Edit`     | `file_path`   | File being edited            |
| `Write`    | `file_path`   | File being written           |
| `Grep`     | `pattern`     | Search pattern               |
| `Glob`     | `pattern`     | Glob pattern                 |
| `WebFetch` | `url`         | URL being fetched            |
| `Task`     | `description` | Task description             |
| `Agent`    | `prompt`      | Agent prompt                 |

Tools not in this table display with no content (header line only).

The section color for all tool display is `colorGreen`.

### Content Truncation

If the extracted content contains more than one line, it is truncated:

```
<first line>
... +N lines
```

Where `N` is `len(lines) - 1`. Single-line content is displayed as-is.

---

## 6. Lorah Status Messages

These are printed by the loop (not by `printMessages`) to mark loop lifecycle events:

| Event            | Label     | Color       | Content                                                |
| ---------------- | --------- | ----------- | ------------------------------------------------------ |
| Loop start       | `"Lorah"` | `colorBlue` | `"Starting loop..."`                                   |
| Loop success     | `"Lorah"` | `colorBlue` | `"Loop completed successfully"`                        |
| First interrupt  | `"Lorah"` | `colorBlue` | `"Received interrupt, stopping after current loop..."` |
| Second interrupt | `"Lorah"` | `colorBlue` | `"Received second interrupt, shutting down..."`        |

Error messages on failed iterations are printed directly to stderr (not via `printSection`):

```
<red>⏺ <bold>Error<reset>
<error message>

Retrying in 5s...

```

---

## 7. Constants

```go
const (
    colorReset = "\033[0m"
    colorGreen = "\033[32m"
    colorBlue  = "\033[34m"
    colorBold  = "\033[1m"
    colorRed   = "\033[31m"

    maxBufferSize = 1024 * 1024 // 1MB buffer for JSON parsing
)
```

These constants are package-level in `internal/loop/constants.go` and shared
across `loop.go`, `claude.go`, and `output.go`. `retryDelay` is also defined
in `constants.go` but is a loop concern — see [run.md](run.md).

---

## 8. Related Specifications

- [run.md](run.md) — loop lifecycle that drives `printMessages` and status messages
- [cli.md](cli.md) — CLI structure and entry point
