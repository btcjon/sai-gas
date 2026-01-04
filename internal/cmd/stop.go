package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/events"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/session"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/townlog"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	stopAll      bool
	stopRig      string
	stopGraceful bool
)

var stopCmd = &cobra.Command{
	Use:     "stop",
	GroupID: GroupServices,
	Short:   "Emergency stop for sessions",
	Long: `Emergency stop command for Gas Town sessions.

Stops all running polecat sessions across rigs. By default, runs pre-shutdown
checks on each session to ensure it's in a safe state (clean git, all commits
pushed, beads synced, no hooked work).

Use --graceful to attempt graceful shutdown (Ctrl-C) before killing.
Note that pre-shutdown checks still run unless --graceful is NOT used.

Examples:
  gt stop --all              # Kill all sessions (with safety checks)
  gt stop --rig wyvern       # Kill sessions in wyvern (with safety checks)
  gt stop --all --graceful   # Graceful shutdown with safety checks

For emergency situations where safety checks must be skipped, modify the
session Manager.Stop() call with a force flag if needed.`,
	RunE: runStop,
}

func init() {
	stopCmd.Flags().BoolVar(&stopAll, "all", false, "Stop all sessions across all rigs")
	stopCmd.Flags().StringVar(&stopRig, "rig", "", "Stop all sessions in a specific rig")
	stopCmd.Flags().BoolVar(&stopGraceful, "graceful", false, "Try graceful shutdown before force kill")
	rootCmd.AddCommand(stopCmd)
}

// StopResult tracks what was stopped.
type StopResult struct {
	Rig       string
	Polecat   string
	SessionID string
	Success   bool
	Error     string
}

// formatMultilineError indents multiline error messages for readability.
func formatMultilineError(errMsg string) string {
	lines := strings.Split(errMsg, "\n")
	var formatted []string
	for _, line := range lines {
		if line != "" {
			formatted = append(formatted, line)
		}
	}
	return strings.Join(formatted, "\n     ")
}

func runStop(cmd *cobra.Command, args []string) error {
	if !stopAll && stopRig == "" {
		return fmt.Errorf("must specify --all or --rig <name>")
	}

	// Find town root
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load rigs config
	rigsConfigPath := filepath.Join(townRoot, "mayor", "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
	if err != nil {
		rigsConfig = &config.RigsConfig{Rigs: make(map[string]config.RigEntry)}
	}

	// Get rigs to stop
	g := git.NewGit(townRoot)
	rigMgr := rig.NewManager(townRoot, rigsConfig, g)
	rigs, err := rigMgr.DiscoverRigs()
	if err != nil {
		return fmt.Errorf("discovering rigs: %w", err)
	}

	// Filter by rig if specified
	if stopRig != "" {
		var filtered []*rig.Rig
		for _, r := range rigs {
			if r.Name == stopRig {
				filtered = append(filtered, r)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("rig '%s' not found", stopRig)
		}
		rigs = filtered
	}

	// Determine force mode
	force := !stopGraceful

	if stopAll {
		fmt.Printf("%s Stopping ALL Gas Town sessions...\n\n",
			style.Bold.Render("🛑"))
	} else {
		fmt.Printf("%s Stopping sessions in rig '%s'...\n\n",
			style.Bold.Render("🛑"), stopRig)
	}

	// When not forcing, warn about pre-shutdown checks
	if !force && stopAll {
		fmt.Printf("%s Pre-shutdown checks will be run for each session\n\n",
			style.Dim.Render("ℹ"))
	}

	// Stop sessions in each rig
	t := tmux.NewTmux()
	var results []StopResult
	stopped := 0

	for _, r := range rigs {
		mgr := session.NewManager(t, r)
		infos, err := mgr.List()
		if err != nil {
			continue
		}

		for _, info := range infos {
			result := StopResult{
				Rig:       r.Name,
				Polecat:   info.Polecat,
				SessionID: info.SessionID,
			}

			// Capture output before stopping (best effort)
			output, _ := mgr.Capture(info.Polecat, 50)

			// Stop the session
			err := mgr.Stop(info.Polecat, force)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				fmt.Printf("  %s [%s] %s:\n",
					style.Dim.Render("✗"),
					r.Name, info.Polecat)
				// Print the full error (may be multi-line with details)
				fmt.Printf("     %s\n", formatMultilineError(err.Error()))
			} else {
				result.Success = true
				stopped++
				fmt.Printf("  %s [%s] %s: stopped\n",
					style.Bold.Render("✓"),
					r.Name, info.Polecat)

				// Log kill event
				agent := fmt.Sprintf("%s/%s", r.Name, info.Polecat)
				logger := townlog.NewLogger(townRoot)
				logger.Log(townlog.EventKill, agent, "gt stop")

				// Log kill event to activity feed
				_ = events.LogFeed(events.TypeKill, "gt", events.KillPayload(r.Name, info.Polecat, "gt stop"))

				// Log captured output (truncated)
				if len(output) > 200 {
					output = output[len(output)-200:]
				}
				if output != "" {
					fmt.Printf("      %s\n", style.Dim.Render("(output captured)"))
				}
			}

			results = append(results, result)
		}
	}

	// Summary
	fmt.Println()
	if stopped == 0 {
		fmt.Println("No active sessions to stop.")
	} else {
		fmt.Printf("%s %d session(s) stopped.\n",
			style.Bold.Render("✓"), stopped)
	}

	// Report failures
	failures := 0
	for _, r := range results {
		if !r.Success {
			failures++
		}
	}
	if failures > 0 {
		fmt.Printf("%s %d session(s) failed to stop.\n",
			style.Dim.Render("⚠"), failures)
	}

	return nil
}
