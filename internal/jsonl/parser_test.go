package jsonl

import (
	"strings"
	"testing"
)

func TestParseUserMessage(t *testing.T) {
	input := `{"type":"user","sessionId":"abc123","timestamp":"2026-02-26T17:59:34.666Z","gitBranch":"lathanx/BUGS-3830","cwd":"/project","message":{"role":"user","content":"Squash these commits"},"uuid":"c50c20b3"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected role=user, got %s", msgs[0].Role)
	}
	if msgs[0].Text != "Squash these commits" {
		t.Errorf("expected text='Squash these commits', got '%s'", msgs[0].Text)
	}
	if msgs[0].GitBranch != "lathanx/BUGS-3830" {
		t.Errorf("expected branch=lathanx/BUGS-3830, got %s", msgs[0].GitBranch)
	}
}

func TestParseAssistantTextMessage(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"lathanx/BUGS-3830","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"Done. The commits have been squashed."}],"stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"2393a604"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "assistant" {
		t.Errorf("expected role=assistant, got %s", msgs[0].Role)
	}
	if msgs[0].Text != "Done. The commits have been squashed." {
		t.Errorf("unexpected text: %s", msgs[0].Text)
	}
	if msgs[0].Usage == nil {
		t.Fatal("expected usage, got nil")
	}
	if msgs[0].Usage.InputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", msgs[0].Usage.InputTokens)
	}
}

func TestParseThinkingBlock(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T17:59:38.084Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"thinking","thinking":"Let me think about this...","signature":"abc"},{"type":"text","text":"Here is my answer."}],"usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"3033050c"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Thinking != "Let me think about this..." {
		t.Errorf("unexpected thinking: %s", msgs[0].Thinking)
	}
	if msgs[0].Text != "Here is my answer." {
		t.Errorf("unexpected text: %s", msgs[0].Text)
	}
}

func TestToolUseOnlyMessagesKept(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T17:59:39.490Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_01","name":"Bash","input":{"command":"git status"}}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"9543104a"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (tool_use with tool summary), got %d", len(msgs))
	}
	if len(msgs[0].ToolCalls) != 1 || msgs[0].ToolCalls[0].Name != "Bash" {
		t.Errorf("expected Bash tool call, got %v", msgs[0].ToolCalls)
	}
	if msgs[0].Text != "" {
		t.Errorf("expected empty text, got %q", msgs[0].Text)
	}
}

func TestSkipThinkingOnlyMessages(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T17:59:39.490Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"thinking","thinking":"Let me think about this..."}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"9543104a"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (thinking only), got %d", len(msgs))
	}
}

func TestSkipWhitespaceOnlyText(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T17:59:39.490Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"  \n  "}],"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"9543104a"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (whitespace-only text), got %d", len(msgs))
	}
}

func TestSkipQueueOperations(t *testing.T) {
	input := `{"type":"queue-operation","operation":"enqueue","timestamp":"2026-02-26T17:59:34.450Z","sessionId":"abc123","content":"Some prompt"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (queue-operation), got %d", len(msgs))
	}
}

func TestSkipMetaMessages(t *testing.T) {
	input := `{"type":"user","sessionId":"abc123","timestamp":"2026-02-26T18:00:08.226Z","gitBranch":"main","cwd":"/project","isMeta":true,"message":{"role":"user","content":"<local-command-caveat>Some caveat</local-command-caveat>"},"uuid":"483a6dbf"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (meta), got %d", len(msgs))
	}
}

func TestSkipSystemInstructions(t *testing.T) {
	input := `{"type":"user","sessionId":"abc123","timestamp":"2026-02-26T17:59:34.666Z","gitBranch":"main","cwd":"/project","message":{"role":"user","content":"<system_instruction>\nYou are working inside Conductor\n</system_instruction>\n\nDo something"},"uuid":"c50c20b3"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (system instruction), got %d", len(msgs))
	}
}

func TestParseMultipleMessages(t *testing.T) {
	lines := []string{
		`{"type":"queue-operation","operation":"enqueue","timestamp":"2026-02-26T17:59:34.450Z","sessionId":"abc123"}`,
		`{"type":"user","sessionId":"abc123","timestamp":"2026-02-26T17:59:34.666Z","gitBranch":"main","cwd":"/project","message":{"role":"user","content":"Hello"},"uuid":"1"}`,
		`{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T17:59:38.084Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","content":[{"type":"text","text":"Hi there!"}],"usage":{"input_tokens":5,"output_tokens":3,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"2"}`,
	}
	input := strings.Join(lines, "\n")

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("first message should be user, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("second message should be assistant, got %s", msgs[1].Role)
	}
}

func TestToolResultsSkipped(t *testing.T) {
	input := `{"type":"user","sessionId":"abc123","timestamp":"2026-02-26T17:59:39.799Z","gitBranch":"main","cwd":"/project","message":{"role":"user","content":[{"tool_use_id":"toolu_01","type":"tool_result","content":"some output","is_error":false}]},"uuid":"2b632c71"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages (tool result), got %d", len(msgs))
	}
}

func TestSkillCallBasic(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_01","name":"Skill","input":{"skill":"commit"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"aaa1"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].SkillCalls) != 1 {
		t.Fatalf("expected 1 skill call, got %d", len(msgs[0].SkillCalls))
	}
	expected := SkillCall{Skill: "commit", Args: ""}
	if msgs[0].SkillCalls[0] != expected {
		t.Errorf("expected skill call %+v, got %+v", expected, msgs[0].SkillCalls[0])
	}
}

func TestSkillCallWithArgs(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_02","name":"Skill","input":{"skill":"review-pr","args":"123"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"aaa2"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].SkillCalls) != 1 {
		t.Fatalf("expected 1 skill call, got %d", len(msgs[0].SkillCalls))
	}
	expected := SkillCall{Skill: "review-pr", Args: "123"}
	if msgs[0].SkillCalls[0] != expected {
		t.Errorf("expected skill call %+v, got %+v", expected, msgs[0].SkillCalls[0])
	}
}

func TestMultipleSkillCalls(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_03","name":"Skill","input":{"skill":"commit"}},{"type":"tool_use","id":"toolu_04","name":"Skill","input":{"skill":"pdf","args":"report.pdf"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"aaa3"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].SkillCalls) != 2 {
		t.Fatalf("expected 2 skill calls, got %d", len(msgs[0].SkillCalls))
	}
	if msgs[0].SkillCalls[0].Skill != "commit" {
		t.Errorf("expected first skill=commit, got %s", msgs[0].SkillCalls[0].Skill)
	}
	if msgs[0].SkillCalls[1].Skill != "pdf" {
		t.Errorf("expected second skill=pdf, got %s", msgs[0].SkillCalls[1].Skill)
	}
	if msgs[0].SkillCalls[1].Args != "report.pdf" {
		t.Errorf("expected second args=report.pdf, got %s", msgs[0].SkillCalls[1].Args)
	}
}

func TestSkillCallMixedWithRegularTools(t *testing.T) {
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_05","name":"Bash","input":{"command":"git status"}},{"type":"tool_use","id":"toolu_06","name":"Skill","input":{"skill":"commit","args":"-m fix"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"aaa4"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	// Should have the Bash tool call in ToolCalls
	if len(msgs[0].ToolCalls) < 1 {
		t.Fatalf("expected at least 1 tool call, got %d", len(msgs[0].ToolCalls))
	}
	foundBash := false
	for _, tc := range msgs[0].ToolCalls {
		if tc.Name == "Bash" {
			foundBash = true
		}
	}
	if !foundBash {
		t.Error("expected Bash in ToolCalls")
	}
	// Should have the Skill in SkillCalls
	if len(msgs[0].SkillCalls) != 1 {
		t.Fatalf("expected 1 skill call, got %d", len(msgs[0].SkillCalls))
	}
	if msgs[0].SkillCalls[0].Skill != "commit" {
		t.Errorf("expected skill=commit, got %s", msgs[0].SkillCalls[0].Skill)
	}
	if msgs[0].SkillCalls[0].Args != "-m fix" {
		t.Errorf("expected args='-m fix', got %s", msgs[0].SkillCalls[0].Args)
	}
}

func TestSkillCallMalformedInput(t *testing.T) {
	// Skill tool_use with input that is not valid JSON for skill extraction —
	// should still appear as a regular ToolCall but not panic or error.
	input := `{"type":"assistant","sessionId":"abc123","timestamp":"2026-02-26T18:00:03.572Z","gitBranch":"main","cwd":"/project","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"toolu_07","name":"Skill","input":"not-a-json-object"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}},"uuid":"aaa5"}`

	msgs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	// Malformed input: should have 0 skill calls (graceful degradation)
	if len(msgs[0].SkillCalls) != 0 {
		t.Errorf("expected 0 skill calls for malformed input, got %d", len(msgs[0].SkillCalls))
	}
}
