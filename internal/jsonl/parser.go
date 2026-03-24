package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ParseFile reads a JSONL session file and returns all parsed messages.
func ParseFile(path string) ([]ParsedMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	return Parse(f)
}

// ParseFileFrom reads a JSONL session file starting from a byte offset.
// Returns parsed messages and the new byte offset (end of file).
func ParseFileFrom(path string, offset int64) ([]ParsedMessage, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, 0, fmt.Errorf("seeking to offset %d: %w", offset, err)
		}
	}

	msgs, err := Parse(f)
	if err != nil {
		return nil, 0, err
	}

	newOffset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, 0, fmt.Errorf("getting file size: %w", err)
	}

	return msgs, newOffset, nil
}

// Parse reads JSONL data from a reader and returns parsed messages.
func Parse(r io.Reader) ([]ParsedMessage, error) {
	var messages []ParsedMessage
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // up to 10MB lines

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // skip malformed lines
		}

		msg := processEntry(entry)
		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	if err := scanner.Err(); err != nil {
		return messages, fmt.Errorf("scanning: %w", err)
	}

	return messages, nil
}

func processEntry(entry Entry) *ParsedMessage {
	// Skip queue operations, file history snapshots, and meta messages
	if entry.Type == "queue-operation" || entry.Type == "file-history-snapshot" {
		return nil
	}
	if entry.IsMeta {
		return nil
	}
	if entry.Message == nil {
		return nil
	}

	ts := parseTimestamp(entry.Timestamp)

	switch entry.Message.Role {
	case "user":
		return processUserEntry(entry, ts)
	case "assistant":
		return processAssistantEntry(entry, ts)
	default:
		return nil
	}
}

func processUserEntry(entry Entry, ts time.Time) *ParsedMessage {
	if entry.Message.Content == nil {
		return nil
	}

	// User content can be a string or an array of tool_result blocks
	if text, ok := ParseContentString(entry.Message.Content); ok {
		// Skip system instructions and local command output
		if strings.Contains(text, "<system_instruction>") ||
			strings.Contains(text, "<local-command-") {
			return nil
		}
		return &ParsedMessage{
			SessionID: entry.SessionID,
			Timestamp: ts,
			Role:      "user",
			GitBranch: entry.GitBranch,
			Cwd:       entry.Cwd,
			Text:      text,
		}
	}

	// Array content = tool results, skip for daily log (tool results are noise)
	return nil
}

func processAssistantEntry(entry Entry, ts time.Time) *ParsedMessage {
	if entry.Message.Content == nil {
		return nil
	}

	blocks, err := ParseContentBlocks(entry.Message.Content)
	if err != nil {
		return nil
	}

	msg := &ParsedMessage{
		SessionID: entry.SessionID,
		Timestamp: ts,
		Role:      "assistant",
		GitBranch: entry.GitBranch,
		Cwd:       entry.Cwd,
		Usage:     entry.Message.Usage,
	}

	var textParts []string
	var thinkingParts []string

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "thinking":
			if block.Thinking != "" {
				thinkingParts = append(thinkingParts, block.Thinking)
			}
		case "tool_use":
			inputStr := ""
			if block.Input != nil {
				inputStr = string(block.Input)
			}
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				Name:  block.Name,
				Input: inputStr,
			})
			if block.Name == "Skill" && block.Input != nil {
				var si struct {
					Skill string `json:"skill"`
					Args  string `json:"args"`
				}
				if json.Unmarshal(block.Input, &si) == nil && si.Skill != "" {
					msg.SkillCalls = append(msg.SkillCalls, SkillCall{
						Skill: si.Skill,
						Args:  si.Args,
					})
				}
			}
		}
	}

	msg.Text = strings.TrimSpace(strings.Join(textParts, "\n"))
	msg.Thinking = strings.Join(thinkingParts, "\n")

	// Skip entries that have neither visible text nor tool calls
	if msg.Text == "" && len(msg.ToolCalls) == 0 {
		return nil
	}

	return msg
}

func parseTimestamp(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}
