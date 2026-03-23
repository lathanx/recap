package cmd

import (
	"testing"
)

func TestRecapRootCommandName(t *testing.T) {
	if rootCmd.Use != "recap" {
		t.Errorf("expected root command name to be %q, got %q", "recap", rootCmd.Use)
	}
}

func TestRecapRootCommandDescription(t *testing.T) {
	if rootCmd.Short == "" {
		t.Fatal("expected root command to have a short description")
	}
	// The description should reference "recap", not "bartlebee"
	want := "recap"
	got := rootCmd.Short
	found := false
	for i := 0; i <= len(got)-len(want); i++ {
		lower := ""
		for _, c := range got[i : i+len(want)] {
			if c >= 'A' && c <= 'Z' {
				lower += string(c + 32)
			} else {
				lower += string(c)
			}
		}
		if lower == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected root command short description to mention %q, got %q", want, got)
	}
}

func TestRecapArchiveCommandRemoved(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "archive" || cmd.Name() == "archive" {
			t.Error("archive command should not exist, but it was found")
		}
	}
}

func TestRecapStatusCommandRemoved(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status" || cmd.Name() == "status" {
			t.Error("status command should not exist, but it was found")
		}
	}
}

func TestRecapExpectedCommandsExist(t *testing.T) {
	expected := []string{"sync", "summarize", "install", "uninstall", "hook", "run"}
	commands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected command %q to exist, but it was not found", name)
		}
	}
}
