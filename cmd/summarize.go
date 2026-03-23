package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lathanx/recap/internal/log"
	"github.com/spf13/cobra"
)

var summarizeDate string

// summarizeSystemPrompt is the system prompt sent to Claude for summarization.
var summarizeSystemPrompt = `You are a log summarizer for a developer's Claude Code sessions. You have access to read files and write files.

Your job:
1. Read the daily log file provided
2. Group the content by branch or topic
3. For each group: note the goal, what was accomplished, key decisions, and remaining work
4. Write a concise summary (2-3 sentences per group)
5. Write the summary to the specified output file

Be concise and focus on what matters: decisions made, problems solved, things learned.`

// summarizeUserPromptFmt is the format string for the user prompt sent to Claude.
// Arguments: logPath, summaryPath, dateStr
var summarizeUserPromptFmt = "Read the daily log at %s. Write a summary to %s. Today's date is %s."

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Generate AI summary for a day's log via claude -p",
	RunE:  runSummarize,
}

func init() {
	summarizeCmd.Flags().StringVar(&summarizeDate, "date", "", "Date to summarize (YYYY-MM-DD, defaults to today)")
	rootCmd.AddCommand(summarizeCmd)
}

type claudeResult struct {
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	DurationMS   int     `json:"duration_ms"`
	NumTurns     int     `json:"num_turns"`
	Usage        struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
}

func runSummarize(cmd *cobra.Command, args []string) error {
	targetDate := time.Now()
	if summarizeDate != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", summarizeDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
	}

	logPath := log.DailyLogPath(targetDate)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("no daily log found at %s", logPath)
	}

	home, _ := os.UserHomeDir()
	recapDir := filepath.Join(home, ".recap")
	summaryPath := strings.TrimSuffix(logPath, ".md") + ".summary.md"
	dateStr := targetDate.Format("2006-01-02")

	systemPrompt := summarizeSystemPrompt

	prompt := fmt.Sprintf(summarizeUserPromptFmt,
		logPath, summaryPath, dateStr,
	)

	fmt.Printf("Generating summary for %s...\n", dateStr)

	// Run claude -p with tool access
	claudeCmd := exec.Command("claude", "-p",
		"--model", "sonnet",
		"--tools", "Read,Write,Glob",
		"--output-format", "json",
		"--no-session-persistence",
		"--system-prompt", systemPrompt,
		prompt,
	)

	// Remove CLAUDECODE env vars to allow nested invocation
	env := os.Environ()
	filteredEnv := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "CLAUDECODE=") &&
			!strings.HasPrefix(e, "CLAUDE_CODE_ENTRYPOINT=") &&
			!strings.HasPrefix(e, "CLAUDE_AGENT_SDK_VERSION=") {
			filteredEnv = append(filteredEnv, e)
		}
	}
	claudeCmd.Env = filteredEnv

	output, err := claudeCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("claude -p failed: %s\n%s", err, string(exitErr.Stderr))
		}
		return fmt.Errorf("claude -p failed: %w", err)
	}

	// Parse JSON response for usage logging
	var result claudeResult
	if err := json.Unmarshal(output, &result); err != nil {
		fmt.Printf("Warning: couldn't parse claude response for usage tracking: %v\n", err)
	} else {
		logUsage(recapDir, "summarize", result)
	}

	fmt.Printf("Summary written to %s\n", summaryPath)
	return nil
}

func logUsage(recapDir, command string, result claudeResult) {
	usagePath := filepath.Join(recapDir, "logs", "usage.log")
	os.MkdirAll(filepath.Dir(usagePath), 0755)

	f, err := os.OpenFile(usagePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	entry := fmt.Sprintf("%s | %-10s | sonnet | in:%-6d out:%-5d cache:%-5d | $%.4f | %d turns | %.1fs\n",
		time.Now().Format("2006-01-02 15:04:05"),
		command,
		result.Usage.InputTokens,
		result.Usage.OutputTokens,
		result.Usage.CacheReadInputTokens,
		result.TotalCostUSD,
		result.NumTurns,
		float64(result.DurationMS)/1000.0,
	)

	f.WriteString(entry)
}
