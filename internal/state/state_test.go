package state

import (
	"os"
	"strings"
	"testing"
)

func TestMarkHookedAndIsHooked(t *testing.T) {
	s := &State{
		Offsets:        make(map[string]int64),
		HookedSessions: make(map[string]bool),
	}

	if s.IsHooked("sess-1") {
		t.Error("expected IsHooked to return false for unknown session")
	}

	s.MarkHooked("sess-1")

	if !s.IsHooked("sess-1") {
		t.Error("expected IsHooked to return true after MarkHooked")
	}
	if s.IsHooked("sess-2") {
		t.Error("expected IsHooked to return false for different session")
	}
}

func TestMarkHookedNilMap(t *testing.T) {
	s := &State{Offsets: make(map[string]int64)}

	// MarkHooked should initialize the map if nil
	s.MarkHooked("sess-1")

	if !s.IsHooked("sess-1") {
		t.Error("expected IsHooked to return true after MarkHooked on nil map")
	}
}

func TestClearHookedSessions(t *testing.T) {
	s := &State{
		Offsets:        make(map[string]int64),
		HookedSessions: map[string]bool{"sess-1": true, "sess-2": true},
	}

	s.ClearHookedSessions()

	if s.IsHooked("sess-1") || s.IsHooked("sess-2") {
		t.Error("expected all hooked sessions to be cleared")
	}
}

func TestStatePathUsesRecapPath(t *testing.T) {
	path := statePath()
	if !strings.Contains(path, ".recap") {
		t.Errorf("statePath() = %q, want path containing .recap", path)
	}
	if strings.Contains(path, ".bartlebee") {
		t.Errorf("statePath() = %q, should not contain .bartlebee", path)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	// Use a temp dir to avoid touching the real state file
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	s, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	s.MarkHooked("sess-abc")
	s.SetOffset("/some/file.jsonl", 42)

	if err := s.Save(); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	s2, err := Load()
	if err != nil {
		t.Fatalf("Load after save error: %v", err)
	}

	if !s2.IsHooked("sess-abc") {
		t.Error("expected hooked session to survive round-trip")
	}
	if s2.GetOffset("/some/file.jsonl") != 42 {
		t.Error("expected offset to survive round-trip")
	}
}
