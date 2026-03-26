package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/lathanx/recap/internal/jsonl"
	"github.com/lathanx/recap/internal/state"
)

func TestFormatToolSummary(t *testing.T) {
	tests := []struct {
		name  string
		calls []jsonl.ToolCall
		want  string
	}{
		{
			name:  "single tool",
			calls: []jsonl.ToolCall{{Name: "Read"}},
			want:  "Read",
		},
		{
			name: "multiple different tools",
			calls: []jsonl.ToolCall{
				{Name: "Read"},
				{Name: "Grep"},
				{Name: "Edit"},
			},
			want: "Read, Grep, Edit",
		},
		{
			name: "repeated tools show count",
			calls: []jsonl.ToolCall{
				{Name: "Read"},
				{Name: "Grep"},
				{Name: "Read"},
				{Name: "Read"},
			},
			want: "Read(3), Grep",
		},
		{
			name: "preserves first-seen order",
			calls: []jsonl.ToolCall{
				{Name: "Grep"},
				{Name: "Read"},
				{Name: "Grep"},
				{Name: "Edit"},
				{Name: "Read"},
			},
			want: "Grep(2), Read(2), Edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolSummary(tt.calls)
			if got != tt.want {
				t.Errorf("formatToolSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndentContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "hello world",
			want:  " hello world",
		},
		{
			name:  "multiline",
			input: "line one\nline two",
			want:  " line one\n line two",
		},
		{
			name:  "empty lines preserved",
			input: "before\n\nafter",
			want:  " before\n\n after",
		},
		{
			name:  "header-like line escaped",
			input: "### 12:00:00 [USER] (session:abc12345 branch:main)",
			want:  " ### 12:00:00 [USER] (session:abc12345 branch:main)",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentContent(tt.input)
			if got != tt.want {
				t.Errorf("indentContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMessageIndentsContent(t *testing.T) {
	msg := jsonl.ParsedMessage{
		SessionID: "abcdefghij",
		Role:      "user",
		Timestamp: time.Date(2026, 3, 12, 14, 30, 0, 0, time.Local),
		GitBranch: "main",
		Text:      "### 15:00:00 [USER] (session:fake1234 branch:main)",
	}

	output := formatMessage(msg)

	// The content line must be indented so it doesn't look like a real header
	if strings.Contains(output, "\n### 15:00:00 [USER]") {
		t.Error("formatMessage should indent content lines so fake headers don't match real header prefix")
	}
	// But the real header should still be there
	if !strings.Contains(output, "\n### 14:30:00 [USER]") {
		t.Error("real header should still be present")
	}
}

func TestFormatMessageNoTicketInfo(t *testing.T) {
	tests := []struct {
		name   string
		branch string
	}{
		{
			name:   "branch with ticket prefix",
			branch: "BUGS-1234/fix-login",
		},
		{
			name:   "branch with ticket only",
			branch: "PROJ-999",
		},
		{
			name:   "plain branch",
			branch: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := jsonl.ParsedMessage{
				SessionID: "abcdefghij",
				Role:      "user",
				Timestamp: time.Date(2026, 3, 12, 14, 30, 0, 0, time.Local),
				GitBranch: tt.branch,
				Text:      "some user message",
			}

			output := formatMessage(msg)

			if strings.Contains(output, "ticket:") {
				t.Errorf("formatMessage output should not contain ticket info, got:\n%s", output)
			}
		})
	}
}

func TestFormatMessageAssistantNoTicketInfo(t *testing.T) {
	msg := jsonl.ParsedMessage{
		SessionID: "abcdefghij",
		Role:      "assistant",
		Timestamp: time.Date(2026, 3, 12, 14, 30, 0, 0, time.Local),
		GitBranch: "BUGS-1234/fix-login",
		Text:      "here is my response",
	}

	output := formatMessage(msg)

	if strings.Contains(output, "ticket:") {
		t.Errorf("assistant formatMessage output should not contain ticket info, got:\n%s", output)
	}
}

func TestFilterMessagesByDateRange(t *testing.T) {
	msgs := []jsonl.ParsedMessage{
		{Timestamp: time.Date(2026, 3, 20, 10, 0, 0, 0, time.Local), Text: "day20"},
		{Timestamp: time.Date(2026, 3, 22, 14, 30, 0, 0, time.Local), Text: "day22"},
		{Timestamp: time.Date(2026, 3, 24, 8, 0, 0, 0, time.Local), Text: "day24"},
		{Timestamp: time.Date(2026, 3, 26, 12, 0, 0, 0, time.Local), Text: "day26"},
	}

	start := time.Date(2026, 3, 21, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)

	got := filterMessagesByDateRange(msgs, start, end)

	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Text != "day22" {
		t.Errorf("first message text = %q, want %q", got[0].Text, "day22")
	}
	if got[1].Text != "day24" {
		t.Errorf("second message text = %q, want %q", got[1].Text, "day24")
	}
}

func TestFilterMessagesByDateRange_EmptyInput(t *testing.T) {
	start := time.Date(2026, 3, 21, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 24, 0, 0, 0, 0, time.Local)

	got := filterMessagesByDateRange(nil, start, end)
	if len(got) != 0 {
		t.Errorf("nil input: expected 0 messages, got %d", len(got))
	}

	got = filterMessagesByDateRange([]jsonl.ParsedMessage{}, start, end)
	if len(got) != 0 {
		t.Errorf("empty input: expected 0 messages, got %d", len(got))
	}
}

func TestDeduplicateHooked(t *testing.T) {
	now := time.Now()
	msgs := []jsonl.ParsedMessage{
		{SessionID: "sess-1", Role: "user", Text: "hello", Timestamp: now},
		{SessionID: "sess-1", Role: "assistant", Text: "first reply", Timestamp: now},
		{SessionID: "sess-1", Role: "user", Text: "followup", Timestamp: now},
		{SessionID: "sess-1", Role: "assistant", Text: "last reply", Timestamp: now},
	}

	st := &state.State{
		Offsets:        make(map[string]int64),
		HookedSessions: map[string]bool{"sess-1": true},
	}

	filtered := deduplicateHooked(msgs, st)

	if len(filtered) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(filtered))
	}
	// The last assistant message ("last reply") should be dropped
	for _, m := range filtered {
		if m.Text == "last reply" {
			t.Error("expected last assistant message to be filtered out")
		}
	}
	// First assistant message should still be present
	found := false
	for _, m := range filtered {
		if m.Text == "first reply" {
			found = true
		}
	}
	if !found {
		t.Error("expected non-last assistant message to be preserved")
	}
}

func TestDeduplicateHookedNoEffect(t *testing.T) {
	now := time.Now()
	msgs := []jsonl.ParsedMessage{
		{SessionID: "sess-2", Role: "user", Text: "hello", Timestamp: now},
		{SessionID: "sess-2", Role: "assistant", Text: "reply", Timestamp: now},
	}

	st := &state.State{
		Offsets:        make(map[string]int64),
		HookedSessions: map[string]bool{"sess-1": true}, // different session
	}

	filtered := deduplicateHooked(msgs, st)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 messages (unhooked session), got %d", len(filtered))
	}
}

func TestDeduplicateHookedMultipleSessions(t *testing.T) {
	now := time.Now()
	msgs := []jsonl.ParsedMessage{
		{SessionID: "sess-1", Role: "assistant", Text: "s1 msg", Timestamp: now},
		{SessionID: "sess-2", Role: "assistant", Text: "s2 msg", Timestamp: now},
	}

	st := &state.State{
		Offsets:        make(map[string]int64),
		HookedSessions: map[string]bool{"sess-1": true}, // only sess-1 hooked
	}

	filtered := deduplicateHooked(msgs, st)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 message, got %d", len(filtered))
	}
	if filtered[0].SessionID != "sess-2" {
		t.Errorf("expected sess-2 to survive, got %s", filtered[0].SessionID)
	}
}
