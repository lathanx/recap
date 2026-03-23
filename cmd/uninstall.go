package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove recap hooks (preserves logs)",
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	// Remove hooks from settings.json
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := removeHooks(settingsPath); err != nil {
		fmt.Printf("Warning: couldn't remove hooks: %v\n", err)
	}

	fmt.Println("\nRecap hooks removed.")
	fmt.Printf("Logs preserved at %s\n", filepath.Join(home, ".recap"))
	return nil
}

func removeHooks(settingsPath string) error {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", settingsPath, err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing %s: %w", settingsPath, err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		fmt.Println("  No hooks found in settings.json")
		return nil
	}

	// Remove only recap hooks (identified by "recap" in the command)
	for event, hookGroups := range hooks {
		groups, ok := hookGroups.([]interface{})
		if !ok {
			continue
		}
		var kept []interface{}
		for _, group := range groups {
			g, ok := group.(map[string]interface{})
			if !ok {
				kept = append(kept, group)
				continue
			}
			innerHooks, ok := g["hooks"].([]interface{})
			if !ok {
				kept = append(kept, group)
				continue
			}
			isRecap := false
			for _, h := range innerHooks {
				hm, ok := h.(map[string]interface{})
				if !ok {
					continue
				}
				command, _ := hm["command"].(string)
				if strings.Contains(command, "recap") {
					isRecap = true
					break
				}
			}
			if !isRecap {
				kept = append(kept, group)
			}
		}
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling: %w", err)
	}

	if err := os.WriteFile(settingsPath, out, 0644); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	fmt.Printf("  Removed recap hooks from %s\n", settingsPath)
	return nil
}
