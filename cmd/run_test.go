package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestRunSubcommandExists(t *testing.T) {
	commands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commands[cmd.Name()] = true
	}
	if !commands["run"] {
		t.Error("expected 'run' subcommand to exist on root command, but it was not found")
	}
}

func TestDailySubcommandDoesNotExist(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "daily" {
			t.Error("'daily' subcommand should not exist (it was renamed to 'run'), but it was found")
		}
	}
}

func TestRunCommandUseField(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
			found = true
			if cmd.Use != "run" {
				t.Errorf("expected run command Use field to be %q, got %q", "run", cmd.Use)
			}
		}
	}
	if !found {
		t.Fatal("run command not found; cannot verify Use field")
	}
}

func TestRunCommandShortDescriptionMentionsSyncAndSummarize(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
			found = true
			short := strings.ToLower(cmd.Short)
			if !strings.Contains(short, "sync") {
				t.Errorf("expected run command Short to mention 'sync', got %q", cmd.Short)
			}
			if !strings.Contains(short, "summarize") {
				t.Errorf("expected run command Short to mention 'summarize', got %q", cmd.Short)
			}
		}
	}
	if !found {
		t.Fatal("run command not found; cannot verify Short description")
	}
}

func TestRunCommandHasDateFlag(t *testing.T) {
	var found bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "run" {
			found = true
			f := cmd.Flags().Lookup("date")
			if f == nil {
				t.Fatal("expected 'run' command to have a --date flag, but it was not found")
			}
			if f.DefValue != "" {
				t.Errorf("expected --date default value to be empty string, got %q", f.DefValue)
			}
		}
	}
	if !found {
		t.Fatal("run command not found; cannot verify --date flag")
	}
}

func TestResolveDateYesterday(t *testing.T) {
	resolved, err := resolveDate("yesterday")
	if err != nil {
		t.Fatalf("resolveDate(\"yesterday\") returned unexpected error: %v", err)
	}
	expected := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	got := resolved.Format("2006-01-02")
	if got != expected {
		t.Errorf("resolveDate(\"yesterday\") = %q, want %q", got, expected)
	}
}

func TestResolveDateExplicit(t *testing.T) {
	resolved, err := resolveDate("2025-11-15")
	if err != nil {
		t.Fatalf("resolveDate(\"2025-11-15\") returned unexpected error: %v", err)
	}
	got := resolved.Format("2006-01-02")
	if got != "2025-11-15" {
		t.Errorf("resolveDate(\"2025-11-15\") = %q, want %q", got, "2025-11-15")
	}
}

func TestResolveDateEmpty(t *testing.T) {
	resolved, err := resolveDate("")
	if err != nil {
		t.Fatalf("resolveDate(\"\") returned unexpected error: %v", err)
	}
	expected := time.Now().Format("2006-01-02")
	got := resolved.Format("2006-01-02")
	if got != expected {
		t.Errorf("resolveDate(\"\") = %q, want today %q", got, expected)
	}
}
