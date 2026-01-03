package claude

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed commands/*.md
var commandsFS embed.FS

// EnsureCommands provisions .claude/commands/ with standard Gas Town slash commands.
// If the directory already has commands, it leaves them untouched.
// This is called when creating crew workspaces to ensure they have /handoff etc.
func EnsureCommands(workDir string) error {
	commandsDir := filepath.Join(workDir, ".claude", "commands")

	// Create .claude/commands directory if needed
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("creating commands directory: %w", err)
	}

	// List all embedded command files
	entries, err := commandsFS.ReadDir("commands")
	if err != nil {
		return fmt.Errorf("reading embedded commands: %w", err)
	}

	// Copy each command file if it doesn't already exist
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		destPath := filepath.Join(commandsDir, entry.Name())

		// Skip if file already exists
		if _, err := os.Stat(destPath); err == nil {
			continue
		}

		// Read embedded content
		content, err := commandsFS.ReadFile(filepath.Join("commands", entry.Name()))
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", entry.Name(), err)
		}

		// Write to destination
		if err := os.WriteFile(destPath, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// CommandFiles returns the list of standard command file names.
// This is used by doctor to check for missing commands.
func CommandFiles() []string {
	entries, err := commandsFS.ReadDir("commands")
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files
}
