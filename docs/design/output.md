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

| Tool Name  | Input Key     | What Is Shown                                                  |
| ---------- | ------------- | -------------------------------------------------------------- |
| `Bash`     | `command`     | Shell command being executed                                   |
| `Read`     | `file_path`   | File being read                                                |
| `Edit`     | `file_path`   | File being edited                                              |
| `Write`    | `file_path`   | File being written                                             |
| `Grep`     | `pattern`     | Search pattern                                                 |
| `Glob`     | `pattern`     | Glob pattern                                                   |
| `WebFetch` | `url`         | URL being fetched                                              |
| `Task*`    | `description` | Task description (prefix match: `TaskCreate`, `TaskGet`, etc.) |
| `Agent`    | `prompt`      | Agent prompt                                                   |

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

| Event            | Label     | Color       | Content                                                  |
| ---------------- | --------- | ----------- | -------------------------------------------------------- |
| Loop start       | `"Lorah"` | `colorBlue` | `"Starting loop N..."` (N = iteration number)            |
| Loop success     | `"Lorah"` | `colorBlue` | `"Loop N completed successfully"` (N = iteration number) |
| First interrupt  | `"Lorah"` | `colorBlue` | `"Received interrupt, stopping after current loop..."`   |
| Second interrupt | `"Lorah"` | `colorBlue` | `"Received second interrupt, shutting down..."`          |

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
in `constants.go` but is a loop concern — see [loop.md](loop.md).

---

## 8. Examples

Each example is one line of stream-JSON input followed by the observable output. Unknown types, malformed JSON, and non-error results produce no output at all.

### 1. Assistant text block

Input:

```json
{
  "type": "assistant",
  "message": { "content": [{ "type": "text", "text": "Reading the file." }] }
}
```

Output (the icon is uncolored because `printSection` is called with an empty color argument):

```
⏺ <bold>Claude<reset>
Reading the file.

```

### 2. Assistant thinking block

Input:

```json
{
  "type": "assistant",
  "message": {
    "content": [{ "type": "thinking", "thinking": "The user wants..." }]
  }
}
```

Output:

```
⏺ <bold>Claude (thinking)<reset>
The user wants...

```

### 3. Tool use: Bash

Input:

```json
{
  "type": "assistant",
  "message": {
    "content": [
      {
        "type": "tool_use",
        "name": "Bash",
        "input": { "command": "go test ./..." }
      }
    ]
  }
}
```

Output:

```
<green>⏺<reset> <bold>Bash<reset>
go test ./...

```

### 4. Tool use: Read (single-line path)

Input:

```json
{
  "type": "assistant",
  "message": {
    "content": [
      {
        "type": "tool_use",
        "name": "Read",
        "input": { "file_path": "internal/loop/loop.go" }
      }
    ]
  }
}
```

Output:

```
<green>⏺<reset> <bold>Read<reset>
internal/loop/loop.go

```

### 5. Tool use: Bash with multi-line command (truncated)

Input:

```json
{
  "type": "assistant",
  "message": {
    "content": [
      {
        "type": "tool_use",
        "name": "Bash",
        "input": { "command": "set -e\ngo build ./...\ngo test ./..." }
      }
    ]
  }
}
```

Output:

```
<green>⏺<reset> <bold>Bash<reset>
set -e
... +2 lines

```

### 6. Tool use: unmapped tool name

Input:

```json
{
  "type": "assistant",
  "message": {
    "content": [
      { "type": "tool_use", "name": "Unknown", "input": { "foo": "bar" } }
    ]
  }
}
```

Output:

```
<green>⏺<reset> <bold>Unknown<reset>

```

### 7. Result message with error

Input:

```json
{ "type": "result", "is_error": true, "result": "command failed: exit 1" }
```

Output:

```
<red>⏺<reset> <bold>Result (error)<reset>
command failed: exit 1

```

### 8. Result message without error

Input:

```json
{ "type": "result", "is_error": false, "result": "ok" }
```

Output: _(nothing — silently skipped)_

### 9. Unknown top-level message type

Input:

```json
{ "type": "system", "message": "ignored" }
```

Output: _(nothing — silently skipped)_

### 10. Malformed JSON line

Input:

```
not valid json
```

Output: _(nothing — silently skipped, scanner continues to the next line)_

---

## 9. Related Specifications

- [loop.md](loop.md) — loop lifecycle that drives `printMessages`, CLI structure and entry point
