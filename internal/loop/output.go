package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// printMessages reads stream-JSON from r and formats it for display.
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
					printSection("Claude", "", text)
				case "thinking":
					thinking, _ := block["thinking"].(string)
					printSection("Claude (thinking)", "", thinking)
				case "tool_use":
					name, _ := block["name"].(string)
					if name == "" {
						continue
					}
					toolName := strings.ToUpper(name[:1]) + name[1:]
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
					if content != "" {
						lines := strings.Split(content, "\n")
						if len(lines) > 1 {
							content = lines[0] + fmt.Sprintf("\n... +%d lines", len(lines)-1)
						}
					}
					printSection(toolName, colorGreen, content)
				}
			}
		case "result":
			isError, _ := msg["is_error"].(bool)
			if isError {
				result, _ := msg["result"].(string)
				printSection("Result (error)", colorRed, result)
			}
		}
	}
}

// printSection outputs a labeled section with optional content.
func printSection(label, color, content string) {
	fmt.Printf("%s⏺%s %s%s%s\n", color, colorReset, colorBold, label, colorReset)
	if content != "" {
		content = strings.TrimSpace(content)
		fmt.Printf("%s\n", content)
	}
	fmt.Println()
}
