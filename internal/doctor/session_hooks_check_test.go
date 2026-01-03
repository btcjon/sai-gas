package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewSessionHooksConsistencyCheck(t *testing.T) {
	check := NewSessionHooksConsistencyCheck()

	if check.Name() != "session-hooks" {
		t.Errorf("expected name 'session-hooks', got %q", check.Name())
	}

	expectedDesc := "Check settings.json files use session-start.sh for session_id passthrough"
	if check.Description() != expectedDesc {
		t.Errorf("unexpected description: %q", check.Description())
	}

	if !check.CanFix() {
		t.Error("expected CanFix to return true")
	}
}

func TestSessionHooksConsistencyCheck_NoSettingsFiles(t *testing.T) {
	tmpDir := t.TempDir()

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	result := check.Run(ctx)

	if result.Status != StatusOK {
		t.Errorf("expected StatusOK when no settings files, got %v", result.Status)
	}
}

func TestSessionHooksConsistencyCheck_ValidSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory with valid settings
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "bash -c '$HOME/.claude/hooks/session-start.sh'",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	result := check.Run(ctx)

	if result.Status != StatusOK {
		t.Errorf("expected StatusOK for valid settings, got %v: %s", result.Status, result.Message)
	}
}

func TestSessionHooksConsistencyCheck_BareGtPrime(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory with bare gt prime
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "gt prime",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	result := check.Run(ctx)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning for bare gt prime, got %v: %s", result.Status, result.Message)
	}

	if len(result.Details) == 0 {
		t.Error("expected details about the issue")
	}
}

func TestSessionHooksConsistencyCheck_BareGtPrimeWithBashC(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory with bash -c 'gt prime'
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "bash -c 'gt prime && something'",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	result := check.Run(ctx)

	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning for bash -c 'gt prime', got %v: %s", result.Status, result.Message)
	}
}

func TestSessionHooksConsistencyCheck_IsBareGtPrime(t *testing.T) {
	check := NewSessionHooksConsistencyCheck()

	tests := []struct {
		cmd      string
		expected bool
	}{
		// Should detect as bare gt prime
		{"gt prime", true},
		{"gt prime && something", true},
		{"bash -c 'gt prime'", true},
		{`bash -c "gt prime"`, true},

		// Should NOT detect as bare gt prime (uses session-start.sh)
		{"bash -c '$HOME/.claude/hooks/session-start.sh'", false},
		{"bash ~/.claude/hooks/session-start.sh", false},
		{"tsx $HOME/.claude/hooks/session-start.sh && gt prime", false},

		// Should NOT detect (doesn't contain gt prime)
		{"echo hello", false},
		{"tsx something.ts", false},
	}

	for _, tt := range tests {
		result := check.isBareGtPrime(tt.cmd)
		if result != tt.expected {
			t.Errorf("isBareGtPrime(%q) = %v, want %v", tt.cmd, result, tt.expected)
		}
	}
}

func TestSessionHooksConsistencyCheck_Fix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .claude directory with bare gt prime
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "gt prime",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	// First run should detect the issue
	result := check.Run(ctx)
	if result.Status != StatusWarning {
		t.Fatalf("expected StatusWarning before fix, got %v", result.Status)
	}

	// Apply fix
	if err := check.Fix(ctx); err != nil {
		t.Fatalf("Fix() error: %v", err)
	}

	// Read the fixed file
	fixedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var fixedSettings map[string]interface{}
	if err := json.Unmarshal(fixedData, &fixedSettings); err != nil {
		t.Fatal(err)
	}

	// Verify the command was updated
	hooks := fixedSettings["hooks"].(map[string]interface{})
	sessionStart := hooks["SessionStart"].([]interface{})
	entry := sessionStart[0].(map[string]interface{})
	hooksList := entry["hooks"].([]interface{})
	hook := hooksList[0].(map[string]interface{})
	command := hook["command"].(string)

	expected := "bash -c '$HOME/.claude/hooks/session-start.sh'"
	if command != expected {
		t.Errorf("expected command %q after fix, got %q", expected, command)
	}

	// Run again should show OK
	check2 := NewSessionHooksConsistencyCheck()
	result2 := check2.Run(ctx)
	if result2.Status != StatusOK {
		t.Errorf("expected StatusOK after fix, got %v: %s", result2.Status, result2.Message)
	}
}

func TestSessionHooksConsistencyCheck_MultipleMixedSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create town-level settings with session-start.sh (valid)
	townClaudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(townClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	validSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "bash -c '$HOME/.claude/hooks/session-start.sh'",
						},
					},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(validSettings, "", "  ")
	os.WriteFile(filepath.Join(townClaudeDir, "settings.json"), data, 0644)

	// Create a rig with invalid settings
	rigDir := filepath.Join(tmpDir, "myrig")
	rigClaudeDir := filepath.Join(rigDir, ".claude")
	if err := os.MkdirAll(rigClaudeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create rig structure so it's detected as a rig
	os.MkdirAll(filepath.Join(rigDir, "crew"), 0755)

	invalidSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "gt prime",
						},
					},
				},
			},
		},
	}

	data2, _ := json.MarshalIndent(invalidSettings, "", "  ")
	os.WriteFile(filepath.Join(rigClaudeDir, "settings.json"), data2, 0644)

	check := NewSessionHooksConsistencyCheck()
	ctx := &CheckContext{TownRoot: tmpDir}

	result := check.Run(ctx)

	// Should find the issue in the rig settings
	if result.Status != StatusWarning {
		t.Errorf("expected StatusWarning for mixed settings, got %v: %s", result.Status, result.Message)
	}

	// Should have exactly 1 issue (the rig one)
	if len(result.Details) != 1 {
		t.Errorf("expected 1 issue detail, got %d: %v", len(result.Details), result.Details)
	}
}
