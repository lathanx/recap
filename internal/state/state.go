package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State tracks processing offsets for JSONL files.
type State struct {
	// Offsets maps file path to byte offset of last processed position.
	Offsets map[string]int64 `json:"offsets"`
	// HookedSessions tracks sessions whose stop hook has already written an entry.
	HookedSessions map[string]bool `json:"hooked_sessions,omitempty"`
}

func statePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".recap", "state.json")
}

// Load reads the state file. Returns empty state if file doesn't exist.
func Load() (*State, error) {
	s := &State{Offsets: make(map[string]int64)}

	data, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	if s.Offsets == nil {
		s.Offsets = make(map[string]int64)
	}
	if s.HookedSessions == nil {
		s.HookedSessions = make(map[string]bool)
	}

	return s, nil
}

// Save writes the state file.
func (s *State) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(statePath()), 0755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	if err := os.WriteFile(statePath(), data, 0644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}

	return nil
}

// GetOffset returns the byte offset for a file, or 0 if not tracked.
func (s *State) GetOffset(path string) int64 {
	return s.Offsets[path]
}

// SetOffset updates the byte offset for a file.
func (s *State) SetOffset(path string, offset int64) {
	s.Offsets[path] = offset
}

// MarkHooked records that a session's stop hook has written an entry.
func (s *State) MarkHooked(sessionID string) {
	if s.HookedSessions == nil {
		s.HookedSessions = make(map[string]bool)
	}
	s.HookedSessions[sessionID] = true
}

// IsHooked returns whether a session's stop hook has already written an entry.
func (s *State) IsHooked(sessionID string) bool {
	return s.HookedSessions[sessionID]
}

// ClearHookedSessions removes all hooked session records.
func (s *State) ClearHookedSessions() {
	s.HookedSessions = make(map[string]bool)
}
