package doctor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SessionHooksConsistencyCheck verifies settings.json files use session-start.sh
// for SessionStart hooks, enabling proper session_id passthrough across all roles.
type SessionHooksConsistencyCheck struct {
	FixableCheck
	inconsistentFiles []settingsInconsistency
	sourceOfTruthPath string
}

type settingsInconsistency struct {
	path       string
	issue      string // "bare_gt_prime" or "missing_session_start"
	hookType   string // "SessionStart"
	command    string // The problematic command
}

// claudeSettings represents the relevant parts of settings.json
type claudeSettings struct {
	Hooks map[string][]hookEntry `json:"hooks"`
}

type hookEntry struct {
	Matcher string   `json:"matcher,omitempty"`
	Hooks   []hook   `json:"hooks"`
}

type hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// NewSessionHooksConsistencyCheck creates a new session hooks consistency check.
func NewSessionHooksConsistencyCheck() *SessionHooksConsistencyCheck {
	return &SessionHooksConsistencyCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "session-hooks",
				CheckDescription: "Check settings.json files use session-start.sh for session_id passthrough",
			},
		},
	}
}

// Run checks all settings.json files for consistent session hook configuration.
func (c *SessionHooksConsistencyCheck) Run(ctx *CheckContext) *CheckResult {
	c.inconsistentFiles = nil
	c.sourceOfTruthPath = ""

	// Collect all settings.json locations to check
	var settingsPaths []string

	// 1. Global settings (typically the source of truth)
	globalSettings := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
	if _, err := os.Stat(globalSettings); err == nil {
		settingsPaths = append(settingsPaths, globalSettings)
		c.sourceOfTruthPath = globalSettings
	}

	// 2. Town root .claude/settings.json
	townSettings := filepath.Join(ctx.TownRoot, ".claude", "settings.json")
	if _, err := os.Stat(townSettings); err == nil {
		settingsPaths = append(settingsPaths, townSettings)
	}

	// 3. Rig root settings (for each rig)
	rigs := findAllRigs(ctx.TownRoot)
	for _, rig := range rigs {
		rigSettings := filepath.Join(rig, ".claude", "settings.json")
		if _, err := os.Stat(rigSettings); err == nil {
			settingsPaths = append(settingsPaths, rigSettings)
		}

		// 4. mayor/rig settings (canonical source for rig)
		mayorRigSettings := filepath.Join(rig, "mayor", "rig", ".claude", "settings.json")
		if _, err := os.Stat(mayorRigSettings); err == nil {
			settingsPaths = append(settingsPaths, mayorRigSettings)
		}

		// 5. Crew worktree settings
		crewDir := filepath.Join(rig, "crew")
		if entries, err := os.ReadDir(crewDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					crewSettings := filepath.Join(crewDir, entry.Name(), ".claude", "settings.json")
					if _, err := os.Stat(crewSettings); err == nil {
						settingsPaths = append(settingsPaths, crewSettings)
					}
				}
			}
		}

		// 6. Polecat worktree settings
		polecatsDir := filepath.Join(rig, "polecats")
		if entries, err := os.ReadDir(polecatsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					polecatSettings := filepath.Join(polecatsDir, entry.Name(), ".claude", "settings.json")
					if _, err := os.Stat(polecatSettings); err == nil {
						settingsPaths = append(settingsPaths, polecatSettings)
					}
				}
			}
		}
	}

	if len(settingsPaths) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No settings.json files found",
		}
	}

	// Check each settings.json for session hook issues
	var issues []settingsInconsistency
	for _, path := range settingsPaths {
		pathIssues := c.checkSettingsFile(path, ctx.TownRoot)
		issues = append(issues, pathIssues...)
	}

	c.inconsistentFiles = issues

	if len(issues) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: fmt.Sprintf("All %d settings.json file(s) use session-start.sh", len(settingsPaths)),
		}
	}

	// Format details
	details := make([]string, len(issues))
	for i, issue := range issues {
		relPath := c.relativePath(issue.path, ctx.TownRoot)
		details[i] = fmt.Sprintf("%s: %s", relPath, issue.issue)
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusWarning,
		Message: fmt.Sprintf("%d settings.json file(s) have inconsistent session hooks", len(issues)),
		Details: details,
		FixHint: "Update SessionStart hooks to use 'bash ~/.claude/hooks/session-start.sh' for session_id passthrough",
	}
}

// checkSettingsFile examines a single settings.json for session hook issues.
func (c *SessionHooksConsistencyCheck) checkSettingsFile(path, townRoot string) []settingsInconsistency {
	var issues []settingsInconsistency

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var settings claudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}

	// Check SessionStart hooks
	sessionStartHooks, ok := settings.Hooks["SessionStart"]
	if !ok {
		// No SessionStart hooks - not necessarily an issue, skip
		return nil
	}

	hasSessionStartSh := false
	bareGtPrimeCommands := []string{}

	for _, entry := range sessionStartHooks {
		for _, h := range entry.Hooks {
			if h.Type != "command" {
				continue
			}

			cmd := h.Command

			// Check if this uses session-start.sh
			if strings.Contains(cmd, "session-start.sh") {
				hasSessionStartSh = true
			}

			// Check for bare 'gt prime' without session-start.sh wrapper
			// Common problematic patterns:
			// - "gt prime"
			// - "gt prime ..."
			// - "bash -c 'gt prime'"
			if c.isBareGtPrime(cmd) {
				bareGtPrimeCommands = append(bareGtPrimeCommands, cmd)
			}
		}
	}

	// Report issues
	if len(bareGtPrimeCommands) > 0 {
		for _, cmd := range bareGtPrimeCommands {
			issues = append(issues, settingsInconsistency{
				path:     path,
				issue:    fmt.Sprintf("uses bare 'gt prime' without session-start.sh wrapper (missing session_id passthrough)"),
				hookType: "SessionStart",
				command:  cmd,
			})
		}
	} else if !hasSessionStartSh && len(sessionStartHooks) > 0 {
		// Has SessionStart hooks but none use session-start.sh
		// This is a softer warning - they might have a custom setup
		issues = append(issues, settingsInconsistency{
			path:     path,
			issue:    "SessionStart hooks don't include session-start.sh (may lack session_id passthrough)",
			hookType: "SessionStart",
			command:  "",
		})
	}

	return issues
}

// isBareGtPrime detects if a command is a bare 'gt prime' call without session-start.sh.
func (c *SessionHooksConsistencyCheck) isBareGtPrime(cmd string) bool {
	// Normalize the command
	normalized := strings.TrimSpace(cmd)

	// Already uses session-start.sh? Not bare.
	if strings.Contains(normalized, "session-start.sh") {
		return false
	}

	// Check various patterns of gt prime
	patterns := []string{
		"gt prime",
		"'gt prime'",
		"\"gt prime\"",
	}

	for _, pattern := range patterns {
		if strings.Contains(normalized, pattern) {
			return true
		}
	}

	return false
}

// relativePath returns a path relative to town root or home for display.
func (c *SessionHooksConsistencyCheck) relativePath(path, townRoot string) string {
	// Try relative to town root first
	if rel, err := filepath.Rel(townRoot, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}

	// Try relative to home
	home := os.Getenv("HOME")
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + rel
	}

	return path
}

// Fix updates settings.json files to use session-start.sh.
// This is a conservative fix that only replaces bare 'gt prime' commands.
func (c *SessionHooksConsistencyCheck) Fix(ctx *CheckContext) error {
	if len(c.inconsistentFiles) == 0 {
		return nil
	}

	var errors []string

	for _, issue := range c.inconsistentFiles {
		// Only fix bare gt prime issues automatically
		if issue.command == "" {
			continue // Skip soft warnings
		}

		err := c.fixSettingsFile(issue.path, issue.command)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", issue.path, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to fix some files: %s", strings.Join(errors, "; "))
	}

	return nil
}

// fixSettingsFile replaces bare 'gt prime' with session-start.sh in a settings file.
func (c *SessionHooksConsistencyCheck) fixSettingsFile(path, oldCommand string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid hooks structure")
	}

	sessionStartRaw, ok := hooks["SessionStart"]
	if !ok {
		return fmt.Errorf("no SessionStart hooks found")
	}

	sessionStart, ok := sessionStartRaw.([]interface{})
	if !ok {
		return fmt.Errorf("invalid SessionStart structure")
	}

	modified := false
	newCommand := "bash -c '$HOME/.claude/hooks/session-start.sh'"

	for i, entryRaw := range sessionStart {
		entry, ok := entryRaw.(map[string]interface{})
		if !ok {
			continue
		}

		hooksRaw, ok := entry["hooks"].([]interface{})
		if !ok {
			continue
		}

		for j, hookRaw := range hooksRaw {
			h, ok := hookRaw.(map[string]interface{})
			if !ok {
				continue
			}

			cmd, ok := h["command"].(string)
			if !ok {
				continue
			}

			if c.isBareGtPrime(cmd) {
				// Replace with session-start.sh
				h["command"] = newCommand
				hooksRaw[j] = h
				modified = true
			}
		}

		entry["hooks"] = hooksRaw
		sessionStart[i] = entry
	}

	if !modified {
		return nil
	}

	hooks["SessionStart"] = sessionStart
	settings["hooks"] = hooks

	// Write back with indentation
	newData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}
