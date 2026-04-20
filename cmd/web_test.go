package cmd

import (
	"testing"
)

func TestWebCommandExists(t *testing.T) {
	commands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commands[cmd.Name()] = true
	}
	if !commands["web"] {
		t.Error("expected 'web' subcommand to exist on root command, but it was not found")
	}
}

func TestWebCommandPortFlag(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "web" {
			found = true
			f := cmd.Flags().Lookup("port")
			if f == nil {
				t.Fatal("expected --port flag on web command, but it was not found")
			}
			if f.DefValue != "8484" {
				t.Errorf("expected --port default to be %q, got %q", "8484", f.DefValue)
			}
			break
		}
	}
	if !found {
		t.Fatal("web command not found; cannot test --port flag")
	}
}
