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

func TestFormatSkillSummary(t *testing.T) {
	tests := []struct {
		name  string
		calls []jsonl.SkillCall
		want  string
	}{
		{
			name:  "single skill",
			calls: []jsonl.SkillCall{{Skill: "commit"}},
			want:  "commit",
		},
		{
			name: "multiple different skills",
			calls: []jsonl.SkillCall{
				{Skill: "commit"},
				{Skill: "review"},
				{Skill: "simplify"},
			},
			want: "commit, review, simplify",
		},
		{
			name: "repeated skills show count",
			calls: []jsonl.SkillCall{
				{Skill: "commit"},
				{Skill: "review"},
				{Skill: "commit"},
				{Skill: "commit"},
			},
			want: "commit(3), review",
		},
		{
			name: "preserves first-seen order",
			calls: []jsonl.SkillCall{
				{Skill: "review"},
				{Skill: "commit"},
				{Skill: "review"},
				{Skill: "simplify"},
				{Skill: "commit"},
			},
			want: "review(2), commit(2), simplify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSkillSummary(tt.calls)
			if got != tt.want {
				t.Errorf("formatSkillSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMessageWithSkills(t *testing.T) {
	msg := jsonl.ParsedMessage{
		SessionID: "abcdefghij",
		Role:      "assistant",
		Timestamp: time.Date(2026, 3, 12, 14, 30, 0, 0, time.Local),
		GitBranch: "main",
		Text:      "done",
		ToolCalls: []jsonl.ToolCall{
			{Name: "Read"},
		},
		SkillCalls: []jsonl.SkillCall{
			{Skill: "commit"},
		},
	}

	output := formatMessage(msg)

	if !strings.Contains(output, "_Skills: commit_") {
		t.Errorf("expected skills line in output, got:\n%s", output)
	}

	// Skills line should appear after Tools line
	toolsIdx := strings.Index(output, "_Tools:")
	skillsIdx := strings.Index(output, "_Skills:")
	if toolsIdx == -1 || skillsIdx == -1 {
		t.Fatalf("expected both Tools and Skills lines, got:\n%s", output)
	}
	if skillsIdx <= toolsIdx {
		t.Errorf("expected Skills line after Tools line, tools at %d, skills at %d", toolsIdx, skillsIdx)
	}
}

func TestFormatMessageWithSkillsNoTools(t *testing.T) {
	msg := jsonl.ParsedMessage{
		SessionID: "abcdefghij",
		Role:      "assistant",
		Timestamp: time.Date(2026, 3, 12, 14, 30, 0, 0, time.Local),
		GitBranch: "main",
		Text:      "done",
		SkillCalls: []jsonl.SkillCall{
			{Skill: "review"},
		},
	}

	output := formatMessage(msg)

	if !strings.Contains(output, "_Skills: review_") {
		t.Errorf("expected skills line in output even without tools, got:\n%s", output)
	}
	if strings.Contains(output, "_Tools:") {
		t.Errorf("expected no tools line, got:\n%s", output)
	}
}

func TestFormatSkillFrequencyTable(t *testing.T) {
	calls := []jsonl.SkillCall{
		{Skill: "commit"},
		{Skill: "review"},
		{Skill: "commit"},
		{Skill: "simplify"},
		{Skill: "commit"},
		{Skill: "review"},
		{Skill: "commit"},
	}

	got := formatSkillFrequencyTable(calls)

	want := `---
## Skill Usage

| Skill | Count |
|-------|-------|
| commit | 4 |
| review | 2 |
| simplify | 1 |
`

	if got != want {
		t.Errorf("formatSkillFrequencyTable() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatSkillFrequencyTableTiesAlphabetical(t *testing.T) {
	calls := []jsonl.SkillCall{
		{Skill: "zebra"},
		{Skill: "alpha"},
		{Skill: "zebra"},
		{Skill: "alpha"},
	}

	got := formatSkillFrequencyTable(calls)

	// Both have count 2, so alphabetical: alpha before zebra
	want := `---
## Skill Usage

| Skill | Count |
|-------|-------|
| alpha | 2 |
| zebra | 2 |
`

	if got != want {
		t.Errorf("formatSkillFrequencyTable() =\n%s\nwant:\n%s", got, want)
	}
}

func TestCollectSkillCalls(t *testing.T) {
	now := time.Now()
	msgs := []jsonl.ParsedMessage{
		{
			SessionID:  "sess-1",
			Role:       "assistant",
			Timestamp:  now,
			SkillCalls: []jsonl.SkillCall{{Skill: "commit", Args: "-m 'fix'"}},
		},
		{
			SessionID: "sess-1",
			Role:      "user",
			Timestamp: now,
			// no skill calls
		},
		{
			SessionID: "sess-2",
			Role:      "assistant",
			Timestamp: now,
			SkillCalls: []jsonl.SkillCall{
				{Skill: "review", Args: "123"},
				{Skill: "commit", Args: ""},
			},
		},
	}

	got := collectSkillCalls(msgs)

	if len(got) != 3 {
		t.Fatalf("expected 3 skill calls, got %d", len(got))
	}
	// Verify order: commit, review, commit
	wantSkills := []string{"commit", "review", "commit"}
	for i, want := range wantSkills {
		if got[i].Skill != want {
			t.Errorf("skill[%d] = %q, want %q", i, got[i].Skill, want)
		}
	}
	// Verify args preserved
	if got[0].Args != "-m 'fix'" {
		t.Errorf("skill[0].Args = %q, want %q", got[0].Args, "-m 'fix'")
	}
}

func TestCollectSkillCallsEmpty(t *testing.T) {
	// nil messages
	got := collectSkillCalls(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(got))
	}

	// messages with no skill calls
	msgs := []jsonl.ParsedMessage{
		{SessionID: "sess-1", Role: "user", Timestamp: time.Now()},
		{SessionID: "sess-1", Role: "assistant", Timestamp: time.Now()},
	}
	got = collectSkillCalls(msgs)
	if len(got) != 0 {
		t.Errorf("expected empty result for messages without skills, got %d", len(got))
	}
}

func TestCollectSkillCallsIntegration(t *testing.T) {
	now := time.Now()
	msgs := []jsonl.ParsedMessage{
		{
			SessionID:  "sess-1",
			Role:       "assistant",
			Timestamp:  now,
			SkillCalls: []jsonl.SkillCall{{Skill: "commit"}, {Skill: "commit"}},
		},
		{
			SessionID:  "sess-2",
			Role:       "assistant",
			Timestamp:  now,
			SkillCalls: []jsonl.SkillCall{{Skill: "review"}},
		},
		{
			SessionID:  "sess-1",
			Role:       "assistant",
			Timestamp:  now,
			SkillCalls: []jsonl.SkillCall{{Skill: "commit"}},
		},
	}

	calls := collectSkillCalls(msgs)
	table := formatSkillFrequencyTable(calls)

	if !strings.Contains(table, "## Skill Usage") {
		t.Error("expected table to contain '## Skill Usage' header")
	}
	if !strings.Contains(table, "| commit | 3 |") {
		t.Errorf("expected commit count of 3, got:\n%s", table)
	}
	if !strings.Contains(table, "| review | 1 |") {
		t.Errorf("expected review count of 1, got:\n%s", table)
	}

	// commit should appear before review (higher count)
	commitIdx := strings.Index(table, "| commit |")
	reviewIdx := strings.Index(table, "| review |")
	if commitIdx > reviewIdx {
		t.Errorf("expected commit before review in table (sorted by count desc), got:\n%s", table)
	}
}

func TestFormatSkillFrequencyTableEmpty(t *testing.T) {
	got := formatSkillFrequencyTable(nil)
	if got != "" {
		t.Errorf("formatSkillFrequencyTable(nil) = %q, want empty string", got)
	}

	got = formatSkillFrequencyTable([]jsonl.SkillCall{})
	if got != "" {
		t.Errorf("formatSkillFrequencyTable([]) = %q, want empty string", got)
	}
}
