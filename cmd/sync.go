package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lathanx/recap/internal/jsonl"
	"github.com/lathanx/recap/internal/log"
	"github.com/lathanx/recap/internal/state"
	"github.com/spf13/cobra"
)

var syncDate string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Batch parse JSONL session files into daily log",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncDate, "date", "", "Date to sync (YYYY-MM-DD, defaults to today)")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	targetDate := time.Now()
	if syncDate != "" {
		var err error
		targetDate, err = time.Parse("2006-01-02", syncDate)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	projectsDir := filepath.Join(home, ".claude", "projects")

	// Load processing state
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Find all JSONL files
	jsonlFiles, err := findJSONLFiles(projectsDir, targetDate)
	if err != nil {
		return fmt.Errorf("finding JSONL files: %w", err)
	}

	if len(jsonlFiles) == 0 {
		fmt.Printf("No session files found for %s\n", targetDate.Format("2006-01-02"))
		return nil
	}

	fmt.Printf("Found %d session files for %s\n", len(jsonlFiles), targetDate.Format("2006-01-02"))

	totalMessages := 0
	for _, path := range jsonlFiles {
		offset := st.GetOffset(path)
		msgs, newOffset, err := jsonl.ParseFileFrom(path, offset)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: error parsing %s: %v\n", filepath.Base(path), err)
			continue
		}

		if len(msgs) == 0 {
			continue
		}

		// Drop last assistant message for sessions already written by the stop hook
		msgs = deduplicateHooked(msgs, st)

		// Write messages to daily log
		for _, msg := range msgs {
			entry := formatMessage(msg)
			if entry == "" {
				continue
			}
			if err := log.AppendToDaily(targetDate, entry); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: error writing log: %v\n", err)
				continue
			}
			totalMessages++
		}

		st.SetOffset(path, newOffset)
	}

	// Clear hooked sessions when syncing a past date (fully processed)
	if syncDate != "" {
		st.ClearHookedSessions()
	}

	// Save state
	if err := st.Save(); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("Synced %d messages to daily log\n", totalMessages)
	return nil
}

func findJSONLFiles(projectsDir string, targetDate time.Time) ([]string, error) {
	var files []string

	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// Check if file was modified within the target date
		modTime := info.ModTime()
		if modTime.After(dayStart) && modTime.Before(dayEnd) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

// indentContent prefixes every non-empty line with a single space.
// This prevents message content from being misinterpreted as a header
// (lines starting with "### ") by downstream parsers.
func indentContent(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = " " + line
		}
	}
	return strings.Join(lines, "\n")
}

func formatMessage(msg jsonl.ParsedMessage) string {
	var b strings.Builder

	sessionShort := msg.SessionID
	if len(sessionShort) > 8 {
		sessionShort = sessionShort[:8]
	}

	switch msg.Role {
	case "user":
		fmt.Fprintf(&b, "\n### %s [USER] (session:%s branch:%s)\n",
			msg.Timestamp.Local().Format("15:04:05"),
			sessionShort,
			msg.GitBranch,
		)
		b.WriteString(indentContent(msg.Text))
		b.WriteString("\n")

	case "assistant":
		fmt.Fprintf(&b, "\n### %s [CLAUDE] (session:%s",
			msg.Timestamp.Local().Format("15:04:05"),
			sessionShort,
		)
		b.WriteString(")\n")
		if len(msg.ToolCalls) > 0 {
			b.WriteString("_Tools: ")
			b.WriteString(formatToolSummary(msg.ToolCalls))
			b.WriteString("_\n")
		}
		if msg.Text != "" {
			b.WriteString(indentContent(msg.Text))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// deduplicateHooked removes the last assistant message per session when
// the stop hook has already written an entry for that session.
func deduplicateHooked(msgs []jsonl.ParsedMessage, st *state.State) []jsonl.ParsedMessage {
	// Find the last assistant message index per session
	lastAssistant := make(map[string]int) // sessionID -> index
	for i, m := range msgs {
		if m.Role == "assistant" {
			lastAssistant[m.SessionID] = i
		}
	}

	// Build filtered list, skipping last assistant msg for hooked sessions
	var filtered []jsonl.ParsedMessage
	for i, m := range msgs {
		if idx, ok := lastAssistant[m.SessionID]; ok && i == idx && st.IsHooked(m.SessionID) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// filterMessagesByDateRange keeps messages whose local date is within [start, end] inclusive.
func filterMessagesByDateRange(msgs []jsonl.ParsedMessage, start, end time.Time) []jsonl.ParsedMessage {
	var filtered []jsonl.ParsedMessage
	startY, startM, startD := start.Date()
	endY, endM, endD := end.Date()
	startDate := time.Date(startY, startM, startD, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(endY, endM, endD, 0, 0, 0, 0, time.UTC)
	for _, msg := range msgs {
		local := msg.Timestamp.Local()
		y, m, d := local.Date()
		msgDate := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		if !msgDate.Before(startDate) && !msgDate.After(endDate) {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

func formatToolSummary(calls []jsonl.ToolCall) string {
	counts := make(map[string]int)
	var order []string
	for _, tc := range calls {
		if counts[tc.Name] == 0 {
			order = append(order, tc.Name)
		}
		counts[tc.Name]++
	}
	var parts []string
	for _, name := range order {
		if counts[name] == 1 {
			parts = append(parts, name)
		} else {
			parts = append(parts, fmt.Sprintf("%s(%d)", name, counts[name]))
		}
	}
	return strings.Join(parts, ", ")
}
