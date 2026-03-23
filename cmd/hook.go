package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lathanx/recap/internal/log"
	"github.com/lathanx/recap/internal/state"
	"github.com/spf13/cobra"
)

type SessionStartInput struct {
	SessionID     string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd           string `json:"cwd"`
	HookEventName string `json:"hook_event_name"`
	Source        string `json:"source"`
}

type StopInput struct {
	SessionID           string `json:"session_id"`
	TranscriptPath      string `json:"transcript_path"`
	Cwd                 string `json:"cwd"`
	HookEventName       string `json:"hook_event_name"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for Claude Code events",
}

var hookSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Handle SessionStart events",
	RunE:  runHookSession,
}

var hookStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Handle Stop events",
	RunE:  runHookStop,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookSessionCmd)
	hookCmd.AddCommand(hookStopCmd)
}

func readStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

func detectGitBranch(cwd string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runHookSession(cmd *cobra.Command, args []string) error {
	data, err := readStdin()
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input SessionStartInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	branch := detectGitBranch(input.Cwd)
	now := time.Now()

	entry := fmt.Sprintf("\n---\n## Session %.8s | %s | %s\n_Started: %s | Branch: %s_\n",
		input.SessionID,
		filepath.Base(input.Cwd),
		input.Cwd,
		now.UTC().Format(time.RFC3339),
		branch,
	)

	return log.AppendToDaily(now, entry)
}

func runHookStop(cmd *cobra.Command, args []string) error {
	data, err := readStdin()
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input StopInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	if input.LastAssistantMessage == "" {
		return nil
	}

	now := time.Now()

	// Truncate very long messages for the real-time log (sync has the full version)
	msg := input.LastAssistantMessage
	if len(msg) > 2000 {
		msg = msg[:2000] + "\n\n_[truncated — full message available via `recap sync`]_"
	}

	entry := fmt.Sprintf("\n### %s [CLAUDE] (session:%.8s)\n%s\n",
		now.Format("15:04:05"),
		input.SessionID,
		msg,
	)

	if err := log.AppendToDaily(now, entry); err != nil {
		return err
	}

	// Mark this session as hooked so sync can skip its last assistant message
	st, err := state.Load()
	if err == nil {
		st.MarkHooked(input.SessionID)
		st.Save()
	}

	return nil
}
