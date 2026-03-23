package cmd

import (
	"strings"
	"testing"
)

func TestSummarizeSystemPrompt_NoArchiveReferences(t *testing.T) {
	lower := strings.ToLower(summarizeSystemPrompt)
	if strings.Contains(lower, "archive") {
		t.Errorf("system prompt should not mention 'archive', got:\n%s", summarizeSystemPrompt)
	}
}

func TestSummarizeSystemPrompt_NoTicketReferences(t *testing.T) {
	lower := strings.ToLower(summarizeSystemPrompt)
	if strings.Contains(lower, "ticket") {
		t.Errorf("system prompt should not mention 'ticket', got:\n%s", summarizeSystemPrompt)
	}
	if strings.Contains(lower, "per-ticket") {
		t.Errorf("system prompt should not mention 'per-ticket', got:\n%s", summarizeSystemPrompt)
	}
}

func TestSummarizeSystemPrompt_MentionsDailySummary(t *testing.T) {
	lower := strings.ToLower(summarizeSystemPrompt)
	if !strings.Contains(lower, "summary") && !strings.Contains(lower, "summarize") && !strings.Contains(lower, "summarizer") {
		t.Errorf("system prompt should mention daily summary/summarize, got:\n%s", summarizeSystemPrompt)
	}
}

func TestSummarizeUserPrompt_NoArchiveReferences(t *testing.T) {
	lower := strings.ToLower(summarizeUserPromptFmt)
	if strings.Contains(lower, "archive") {
		t.Errorf("user prompt should not mention 'archive', got:\n%s", summarizeUserPromptFmt)
	}
}

func TestSummarizeUserPrompt_NoPerTicketReferences(t *testing.T) {
	lower := strings.ToLower(summarizeUserPromptFmt)
	if strings.Contains(lower, "ticket") {
		t.Errorf("user prompt should not mention 'ticket', got:\n%s", summarizeUserPromptFmt)
	}
}
