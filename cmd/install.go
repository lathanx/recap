package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Set up recap hooks and directories",
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	recapDir := filepath.Join(home, ".recap")

	// Create directory structure
	dirs := []string{
		filepath.Join(recapDir, "logs", "daily"),
		filepath.Join(recapDir, "archive"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		fmt.Printf("  Created %s\n", dir)
	}

	// Find the recap binary path
	binaryPath, err := findBinary()
	if err != nil {
		return fmt.Errorf("finding recap binary: %w", err)
	}
	fmt.Printf("  Binary: %s\n", binaryPath)

	// Register hooks in ~/.claude/settings.json
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := registerHooks(settingsPath, binaryPath); err != nil {
		return fmt.Errorf("registering hooks: %w", err)
	}

	fmt.Println("\nRecap installed successfully!")
	fmt.Println("Hooks are now active for all Claude Code sessions.")
	return nil
}

func findBinary() (string, error) {
	// First try searching PATH for "recap"
	path, err := exec.LookPath("recap")
	if err == nil {
		return path, nil
	}

	// Fall back to the running binary's path
	exe, err := os.Executable()
	if err == nil {
		abs, err := filepath.Abs(exe)
		if err == nil {
			return abs, nil
		}
	}

	return "", fmt.Errorf("recap not found in PATH; install the binary first")
}

func registerHooks(settingsPath, binaryPath string) error {
	// Read existing settings
	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", settingsPath, err)
		}
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parsing %s: %w", settingsPath, err)
		}
	}

	// Build hook configuration
	hooks := map[string]interface{}{
		"SessionStart": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": binaryPath + " hook session",
						"async":   true,
					},
				},
			},
		},
		"Stop": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": binaryPath + " hook stop",
						"async":   true,
					},
				},
			},
		},
	}

	// Merge with existing hooks (preserve non-recap hooks)
	existingHooks, ok := settings["hooks"].(map[string]interface{})
	if ok {
		for event, hookGroups := range hooks {
			existingHooks[event] = hookGroups
		}
		settings["hooks"] = existingHooks
	} else {
		settings["hooks"] = hooks
	}

	// Write back
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", settingsPath, err)
	}

	fmt.Printf("  Registered hooks in %s\n", settingsPath)
	return nil
}
