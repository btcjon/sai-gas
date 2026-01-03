package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/claude"
)

// CrewCommandsCheck validates that crew workspaces have .claude/commands/ provisioned.
// Missing commands means crew workers don't have access to /handoff and other standard commands.
type CrewCommandsCheck struct {
	FixableCheck
	missingCommands []missingCommand // Cached during Run for use in Fix
}

type missingCommand struct {
	crewPath string
	rigName  string
	crewName string
	missing  []string // List of missing command files
}

// NewCrewCommandsCheck creates a new crew commands check.
func NewCrewCommandsCheck() *CrewCommandsCheck {
	return &CrewCommandsCheck{
		FixableCheck: FixableCheck{
			BaseCheck: BaseCheck{
				CheckName:        "crew-commands",
				CheckDescription: "Validate crew workspaces have .claude/commands/ provisioned",
			},
		},
	}
}

// Run checks all crew workspaces for missing .claude/commands/.
func (c *CrewCommandsCheck) Run(ctx *CheckContext) *CheckResult {
	c.missingCommands = nil

	crewDirs := c.findAllCrewDirs(ctx.TownRoot)
	if len(crewDirs) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No crew workspaces found",
		}
	}

	// Get expected command files from the embedded templates
	expectedCommands := claude.CommandFiles()
	if len(expectedCommands) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: "No standard commands defined",
		}
	}

	var validCount int
	var details []string

	for _, cd := range crewDirs {
		commandsDir := filepath.Join(cd.path, ".claude", "commands")

		// Check which command files are missing
		var missing []string
		for _, cmdFile := range expectedCommands {
			cmdPath := filepath.Join(commandsDir, cmdFile)
			if _, err := os.Stat(cmdPath); os.IsNotExist(err) {
				missing = append(missing, cmdFile)
			}
		}

		if len(missing) > 0 {
			c.missingCommands = append(c.missingCommands, missingCommand{
				crewPath: cd.path,
				rigName:  cd.rigName,
				crewName: cd.crewName,
				missing:  missing,
			})
			details = append(details, fmt.Sprintf("%s/%s: missing %s",
				cd.rigName, cd.crewName, strings.Join(missing, ", ")))
		} else {
			validCount++
		}
	}

	if len(c.missingCommands) == 0 {
		return &CheckResult{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: fmt.Sprintf("All %d crew workspaces have commands provisioned", validCount),
		}
	}

	return &CheckResult{
		Name:    c.Name(),
		Status:  StatusWarning,
		Message: fmt.Sprintf("%d crew workspace(s) missing .claude/commands/", len(c.missingCommands)),
		Details: details,
		FixHint: "Run 'gt doctor --fix' to provision missing commands",
	}
}

// Fix provisions missing commands in crew workspaces.
func (c *CrewCommandsCheck) Fix(ctx *CheckContext) error {
	if len(c.missingCommands) == 0 {
		return nil
	}

	var lastErr error
	for _, mc := range c.missingCommands {
		if err := claude.EnsureCommands(mc.crewPath); err != nil {
			lastErr = fmt.Errorf("%s/%s: %w", mc.rigName, mc.crewName, err)
			continue
		}
	}

	return lastErr
}

type crewDirInfo struct {
	path     string
	rigName  string
	crewName string
}

// findAllCrewDirs finds all crew directories in the workspace.
func (c *CrewCommandsCheck) findAllCrewDirs(townRoot string) []crewDirInfo {
	var dirs []crewDirInfo

	entries, err := os.ReadDir(townRoot)
	if err != nil {
		return dirs
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "mayor" {
			continue
		}

		rigName := entry.Name()
		crewPath := filepath.Join(townRoot, rigName, "crew")

		crewEntries, err := os.ReadDir(crewPath)
		if err != nil {
			continue
		}

		for _, crew := range crewEntries {
			if !crew.IsDir() || strings.HasPrefix(crew.Name(), ".") {
				continue
			}
			dirs = append(dirs, crewDirInfo{
				path:     filepath.Join(crewPath, crew.Name()),
				rigName:  rigName,
				crewName: crew.Name(),
			})
		}
	}

	return dirs
}
