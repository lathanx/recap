package jsonl

import (
	"encoding/json"
	"time"
)

// Entry is a single line in a Claude Code JSONL session file.
type Entry struct {
	Type      string    `json:"type"` // "user", "assistant", "queue-operation", "file-history-snapshot"
	SessionID string    `json:"sessionId"`
	Timestamp string    `json:"timestamp"`
	UUID      string    `json:"uuid"`
	ParentUUID *string  `json:"parentUuid"`
	Cwd       string    `json:"cwd"`
	GitBranch string    `json:"gitBranch"`
	Version   string    `json:"version"`
	IsMeta    bool      `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Message   *Message  `json:"message"`

	// queue-operation fields
	Operation string `json:"operation"` // "enqueue", "dequeue"
	Content   string `json:"content"`   // for queue-operation enqueue (user's prompt)
}

// Message is the API message payload within an entry.
type Message struct {
	Role       string          `json:"role"` // "user", "assistant"
	Model      string          `json:"model"`
	Content    json.RawMessage `json:"content"` // string or []ContentBlock
	StopReason *string         `json:"stop_reason"`
	Usage      *Usage          `json:"usage"`
}

// Usage tracks token consumption for an assistant message.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ContentBlock is a single block in an assistant message's content array.
type ContentBlock struct {
	Type     string          `json:"type"` // "text", "thinking", "tool_use", "tool_result"
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	Name     string          `json:"name,omitempty"`     // tool_use: tool name
	ID       string          `json:"id,omitempty"`       // tool_use: tool use ID
	Input    json.RawMessage `json:"input,omitempty"`    // tool_use: tool input
	Content  json.RawMessage `json:"content,omitempty"`  // tool_result: result content
	IsError  bool            `json:"is_error,omitempty"` // tool_result
}

// ParsedMessage is a processed message ready for log output.
type ParsedMessage struct {
	SessionID string
	Timestamp time.Time
	Role      string // "user", "assistant"
	GitBranch string
	Cwd       string
	Text      string   // the visible text content
	Thinking  string   // thinking/reasoning content (assistant only)
	ToolCalls []ToolCall
	Usage     *Usage
	IsMeta    bool
}

// ToolCall represents a tool invocation by the assistant.
type ToolCall struct {
	Name  string
	Input string // serialized JSON
}

// ParseContentString attempts to parse message content as a plain string.
// Returns the string and true if successful, or empty and false if it's an array.
func ParseContentString(raw json.RawMessage) (string, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, true
	}
	return "", false
}

// ParseContentBlocks parses message content as an array of content blocks.
func ParseContentBlocks(raw json.RawMessage) ([]ContentBlock, error) {
	var blocks []ContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}
