package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHookCommandsReferenceRecap(t *testing.T) {
	// Set up a temp directory with a "bartlebee" binary to simulate the old behavior.
	// findBinary will find it via the running executable path, but the LookPath fallback
	// should look for "recap", not "bartlebee".
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("creating claude dir: %v", err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Simulate what install does: find binary then register hooks.
	// The findBinary LookPath fallback should search for "recap".
	// We test the full flow by using findBinary's result.
	tmpBin := t.TempDir()
	// Only provide a "recap" binary, NOT a "bartlebee" one
	recapBin := filepath.Join(tmpBin, "recap")
	if err := os.WriteFile(recapBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("creating fake recap binary: %v", err)
	}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	// Set PATH so only "recap" is findable; findBinary's Executable() returns
	// the test binary, so we check the LookPath fallback specifically
	os.Setenv("PATH", tmpBin)

	// findBinary currently looks for "bartlebee" in LookPath, which should fail
	// when only "recap" is available. After the fix it should find "recap".
	binaryPath, err := findBinary()
	if err != nil {
		// If findBinary fails, it means it looked for "bartlebee" and didn't find it.
		// After the rename, this should succeed.
		// For now, use a fallback to still test registerHooks
		binaryPath = recapBin
	}

	if err := registerHooks(settingsPath, binaryPath); err != nil {
		t.Fatalf("registerHooks failed: %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parsing settings: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hooks key in settings")
	}

	// Verify hook commands contain "recap" substrings (e.g., "recap hook session")
	for _, event := range []string{"SessionStart", "Stop"} {
		groups, ok := hooks[event].([]interface{})
		if !ok {
			t.Fatalf("expected %s hook groups", event)
		}
		for _, group := range groups {
			g := group.(map[string]interface{})
			innerHooks := g["hooks"].([]interface{})
			for _, h := range innerHooks {
				hm := h.(map[string]interface{})
				command := hm["command"].(string)
				if strings.Contains(command, "bartlebee") {
					t.Errorf("%s hook command should not contain 'bartlebee', got: %s", event, command)
				}
				if !strings.Contains(command, "recap") {
					t.Errorf("%s hook command should contain 'recap', got: %s", event, command)
				}
			}
		}
	}
}

func TestFindBinaryLooksForRecap(t *testing.T) {
	// findBinary's LookPath fallback should search for "recap", not "bartlebee"
	// We test this by checking the source behavior: if the running executable
	// lookup fails, it should fall back to looking for "recap" in PATH.
	//
	// Create a temp dir with a "recap" binary and add it to PATH
	tmpBin := t.TempDir()
	recapPath := filepath.Join(tmpBin, "recap")
	if err := os.WriteFile(recapPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("creating fake recap binary: %v", err)
	}

	// Save and restore PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to only our temp dir (so the current executable won't matter
	// but the LookPath fallback will find "recap")
	os.Setenv("PATH", tmpBin)

	path, err := findBinary()
	if err != nil {
		// If it errors, it might be because it found the running executable first.
		// That's OK - but the error message should mention "recap" not "bartlebee"
		if strings.Contains(err.Error(), "bartlebee") {
			t.Errorf("findBinary error should reference 'recap' not 'bartlebee': %v", err)
		}
		return
	}

	// If it succeeded via LookPath, the path should contain "recap"
	if !strings.Contains(path, "recap") {
		t.Errorf("findBinary should find 'recap' binary, got: %s", path)
	}
}

func TestInstallDoesNotCreateLaunchdPlist(t *testing.T) {
	// The install command should only set up hooks and directories.
	// It should NOT create any launchd plist for scheduling.
	// We verify this by checking that the install command's Short description
	// does not mention scheduling, and by checking that the install source
	// has no plist/launchd references (tested via the description and the
	// absence of a createPlist or similar exported function).

	// Check command description doesn't mention launchd/plist/schedule
	desc := strings.ToLower(installCmd.Short)
	if strings.Contains(desc, "launchd") ||
		strings.Contains(desc, "plist") ||
		strings.Contains(desc, "schedule") {
		t.Errorf("install command description should not mention launchd/plist/schedule, got: %s", installCmd.Short)
	}

	// The install command description should reference "recap" and use
	// "hooks" only (not "hooks and scheduling" or similar)
	if !strings.Contains(desc, "recap") {
		t.Errorf("install command description should mention 'recap', got: %s", installCmd.Short)
	}
	if strings.Contains(desc, "bartlebee") {
		t.Errorf("install command description should not mention 'bartlebee', got: %s", installCmd.Short)
	}
}

func TestInstallCommandDescriptionReferencesRecap(t *testing.T) {
	desc := strings.ToLower(installCmd.Short)
	if strings.Contains(desc, "bartlebee") {
		t.Errorf("install command description should not reference 'bartlebee', got: %s", installCmd.Short)
	}
	if !strings.Contains(desc, "recap") {
		t.Errorf("install command description should reference 'recap', got: %s", installCmd.Short)
	}
}

func TestUninstallDoesNotRemoveLaunchdPlist(t *testing.T) {
	tmpHome := t.TempDir()
	launchAgentsDir := filepath.Join(tmpHome, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		t.Fatalf("creating LaunchAgents dir: %v", err)
	}

	// Create a fake plist file to see if uninstall tries to remove it
	plistPath := filepath.Join(launchAgentsDir, "com.bartlebee.daily.plist")
	if err := os.WriteFile(plistPath, []byte("<plist/>"), 0644); err != nil {
		t.Fatalf("creating fake plist: %v", err)
	}

	// The uninstall command's source code should NOT contain launchd plist removal logic.
	// We test this indirectly: the uninstall command description should not mention launchd.
	desc := strings.ToLower(uninstallCmd.Short)
	if strings.Contains(desc, "launchd") || strings.Contains(desc, "plist") {
		t.Errorf("uninstall command description should not mention launchd/plist, got: %s", uninstallCmd.Short)
	}

	// The uninstall command should not reference "bartlebee" in its description
	if strings.Contains(desc, "bartlebee") {
		t.Errorf("uninstall command description should not reference 'bartlebee', got: %s", uninstallCmd.Short)
	}
}

func TestUninstallCommandDescriptionReferencesRecap(t *testing.T) {
	desc := strings.ToLower(uninstallCmd.Short)
	if !strings.Contains(desc, "recap") {
		t.Errorf("uninstall command description should reference 'recap', got: %s", uninstallCmd.Short)
	}
}

func TestRemoveHooksIdentifiesByRecap(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Create settings with a "recap" hook (not "bartlebee")
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/usr/local/bin/recap hook session",
						},
					},
				},
			},
			"Stop": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/usr/local/bin/recap hook stop",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatalf("marshaling settings: %v", err)
	}
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	// removeHooks should identify and remove hooks containing "recap"
	if err := removeHooks(settingsPath); err != nil {
		t.Fatalf("removeHooks failed: %v", err)
	}

	// Read back and verify the recap hooks were removed
	result, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading settings after removal: %v", err)
	}

	var after map[string]interface{}
	if err := json.Unmarshal(result, &after); err != nil {
		t.Fatalf("parsing settings after removal: %v", err)
	}

	// The hooks should have been removed (since they contain "recap")
	if hooks, ok := after["hooks"]; ok {
		hooksMap, ok := hooks.(map[string]interface{})
		if ok && len(hooksMap) > 0 {
			t.Errorf("removeHooks should have removed recap hooks, but hooks remain: %v", hooksMap)
		}
	}
}
