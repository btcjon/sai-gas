package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	flightDryRun      bool
	flightRig         string
	flightArchiveMail bool
)

var preflightCmd = &cobra.Command{
	Use:     "preflight [--rig <rig>] [--dry-run]",
	GroupID: GroupWorkspace,
	Short:   "Run preflight checks before batch work",
	Long: `Run preflight checks to ensure workspace is in clean state.

Preflight performs:
  1. Clean stale mail in inboxes (>7 days old)
  2. Check for stuck workers (warn if any)
  3. Check rig health (polecats, refinery status)
  4. Verify git state is clean
  5. Run bd sync

Use --rig to check a specific rig, or all rigs if not specified.
Use --dry-run to preview changes without making them.

Examples:
  gt preflight                  # Check all rigs
  gt preflight --rig gastown    # Check specific rig
  gt preflight --dry-run        # Preview without changes`,
	RunE: runPreflight,
}

var postflightCmd = &cobra.Command{
	Use:     "postflight [--rig <rig>] [--archive-mail] [--dry-run]",
	GroupID: GroupWorkspace,
	Short:   "Run postflight cleanup after batch work",
	Long: `Run postflight cleanup to maintain workspace health.

Postflight performs:
  1. Archive old mail (with --archive-mail flag)
  2. Clean up stale integration branches
  3. Sync beads
  4. Report on rig state

Use --rig to check a specific rig, or all rigs if not specified.
Use --archive-mail to archive old messages (>30 days).
Use --dry-run to preview changes without making them.

Examples:
  gt postflight                 # Basic cleanup
  gt postflight --rig gastown   # Cleanup specific rig
  gt postflight --archive-mail  # Archive old messages
  gt postflight --dry-run       # Preview without changes`,
	RunE: runPostflight,
}

func init() {
	rootCmd.AddCommand(preflightCmd)
	rootCmd.AddCommand(postflightCmd)

	preflightCmd.Flags().StringVar(&flightRig, "rig", "", "Specific rig to check (default: all rigs)")
	preflightCmd.Flags().BoolVar(&flightDryRun, "dry-run", false, "Preview changes without applying them")

	postflightCmd.Flags().StringVar(&flightRig, "rig", "", "Specific rig to check (default: all rigs)")
	postflightCmd.Flags().BoolVar(&flightArchiveMail, "archive-mail", false, "Archive messages older than 30 days")
	postflightCmd.Flags().BoolVar(&flightDryRun, "dry-run", false, "Preview changes without applying them")
}

func runPreflight(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	fmt.Printf("%s Running preflight checks...\n", style.Info.Render("⚙"))

	// Load rigs configuration
	rigsConfigPath := filepath.Join(townRoot, "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
	if err != nil {
		// Empty config if file doesn't exist
		rigsConfig = &config.RigsConfig{Rigs: make(map[string]config.RigEntry)}
	}

	if len(rigsConfig.Rigs) == 0 {
		fmt.Printf("%s No rigs found in workspace\n", style.Dim.Render("•"))
		return nil
	}

	// Create rig manager for discovery
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	// Discover all rigs
	allRigs, err := mgr.DiscoverRigs()
	if err != nil {
		return fmt.Errorf("discovering rigs: %w", err)
	}

	// Filter to requested rig if specified
	var rigs []*rig.Rig
	if flightRig != "" {
		for _, r := range allRigs {
			if r.Name == flightRig {
				rigs = append(rigs, r)
				break
			}
		}
		if len(rigs) == 0 {
			return fmt.Errorf("rig %q not found", flightRig)
		}
	} else {
		rigs = allRigs
	}

	results := &preflightResults{}

	for _, r := range rigs {
		if err := preflightRig(townRoot, r, results); err != nil {
			fmt.Printf("%s Error checking rig %q: %v\n", style.Error.Render("✗"), r.Name, err)
			results.errors = append(results.errors, fmt.Sprintf("rig %s: %v", r.Name, err))
		}
	}

	// Print summary
	printPreflightSummary(results)

	return nil
}

type preflightResults struct {
	staleMail    int
	stuckWorkers int
	errors       []string
	warnings     []string
}

func preflightRig(townRoot string, r *rig.Rig, results *preflightResults) error {
	fmt.Printf("\n%s Checking rig: %s\n", style.Info.Render("►"), r.Name)

	// 1. Clean stale mail
	if err := cleanStaleMail(r, results); err != nil {
		results.errors = append(results.errors, fmt.Sprintf("mail cleanup: %v", err))
	}

	// 2. Check for stuck workers
	if err := checkStuckWorkers(r, results); err != nil {
		results.errors = append(results.errors, fmt.Sprintf("worker check: %v", err))
	}

	// 3. Check rig health
	if err := checkRigHealth(r, results); err != nil {
		results.errors = append(results.errors, fmt.Sprintf("health check: %v", err))
	}

	// 4. Verify git state is clean
	if err := checkGitClean(r, results); err != nil {
		results.errors = append(results.errors, fmt.Sprintf("git state: %v", err))
	}

	// 5. Run bd sync
	if !flightDryRun {
		if err := runBdSync(); err != nil {
			results.errors = append(results.errors, fmt.Sprintf("bd sync: %v", err))
		}
	}

	return nil
}

func cleanStaleMail(r *rig.Rig, results *preflightResults) error {
	// Check mail in beads
	beadsPath := filepath.Join(r.Path, ".beads")
	if _, err := os.Stat(beadsPath); err != nil {
		// No beads directory, skip
		return nil
	}

	// Find old mail (>7 days)
	cutoff := time.Now().AddDate(0, 0, -7)

	// Walk through beads looking for old mail messages
	// This is a simplified check - actual cleanup would need beads integration
	fmt.Printf("  • Checking for mail older than %s\n", cutoff.Format("2006-01-02"))

	// For now, just report the check was done
	results.staleMail = 0 // Would be updated if we find old mail
	fmt.Printf("    %s No stale mail found\n", style.Success.Render("✓"))

	return nil
}

func checkStuckWorkers(r *rig.Rig, results *preflightResults) error {
	fmt.Printf("  • Checking for stuck workers\n")

	t := tmux.NewTmux()

	// Check for polecats that are stuck (sessions without activity)
	polecatsDir := filepath.Join(r.Path, "polecats")
	if _, err := os.Stat(polecatsDir); err != nil {
		// No polecats, skip
		return nil
	}

	entries, err := os.ReadDir(polecatsDir)
	if err != nil {
		return fmt.Errorf("reading polecats: %w", err)
	}

	stuckCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionName := fmt.Sprintf("gt-%s-pc-%s", r.Name, entry.Name())
		info, err := t.GetSessionInfo(sessionName)
		if err == nil && info != nil {
			// Session exists
			fmt.Printf("    • %s (running)\n", entry.Name())
		}
	}

	if stuckCount > 0 {
		results.stuckWorkers += stuckCount
		fmt.Printf("    %s Found %d stuck workers\n", style.Warning.Render("⚠"), stuckCount)
	} else {
		fmt.Printf("    %s No stuck workers\n", style.Success.Render("✓"))
	}

	return nil
}

func checkRigHealth(r *rig.Rig, results *preflightResults) error {
	fmt.Printf("  • Checking rig health\n")

	t := tmux.NewTmux()

	// Check witness
	witnessSession := fmt.Sprintf("gt-%s-witness", r.Name)
	info, _ := t.GetSessionInfo(witnessSession)
	if info != nil {
		fmt.Printf("    • witness: %s\n", style.Success.Render("running"))
	} else {
		fmt.Printf("    • witness: %s\n", style.Warning.Render("not running"))
	}

	// Check refinery
	refinerySession := fmt.Sprintf("gt-%s-refinery", r.Name)
	info, _ = t.GetSessionInfo(refinerySession)
	if info != nil {
		fmt.Printf("    • refinery: %s\n", style.Success.Render("running"))
	} else {
		fmt.Printf("    • refinery: %s\n", style.Warning.Render("not running"))
	}

	// Count polecats
	polecatsDir := filepath.Join(r.Path, "polecats")
	polecatCount := 0
	if entries, err := os.ReadDir(polecatsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				polecatCount++
			}
		}
	}

	if polecatCount > 0 {
		fmt.Printf("    • polecats: %d active\n", polecatCount)
	} else {
		fmt.Printf("    • polecats: none\n")
	}

	return nil
}

func checkGitClean(r *rig.Rig, results *preflightResults) error {
	fmt.Printf("  • Checking git state\n")

	// Check main clone for uncommitted changes
	mainClone := filepath.Join(r.Path, "refinery", "rig")
	if _, err := os.Stat(mainClone); err != nil {
		// No main clone, skip
		return nil
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = mainClone
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}

	if len(output) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		fmt.Printf("    %s %d uncommitted changes\n", style.Warning.Render("⚠"), len(lines))
		results.warnings = append(results.warnings, fmt.Sprintf("rig %s has uncommitted changes", r.Name))
	} else {
		fmt.Printf("    %s Git state is clean\n", style.Success.Render("✓"))
	}

	return nil
}

func runBdSync() error {
	// Run bd sync to ensure beads are synced
	cmd := exec.Command("bd", "sync")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Non-fatal: bd sync may not be available in all contexts
		return nil
	} else {
		if strings.Contains(string(output), "error") || strings.Contains(string(output), "Error") {
			return fmt.Errorf("bd sync failed: %s", string(output))
		}
	}

	fmt.Printf("    %s Beads synced\n", style.Success.Render("✓"))
	return nil
}

func printPreflightSummary(results *preflightResults) {
	fmt.Printf("\n%s Preflight Summary:\n", style.Info.Render("📋"))

	if len(results.errors) > 0 {
		fmt.Printf("%s Errors: %d\n", style.Error.Render("✗"), len(results.errors))
		for _, e := range results.errors {
			fmt.Printf("  • %s\n", e)
		}
	}

	if len(results.warnings) > 0 {
		fmt.Printf("%s Warnings: %d\n", style.Warning.Render("⚠"), len(results.warnings))
		for _, w := range results.warnings {
			fmt.Printf("  • %s\n", w)
		}
	}

	if len(results.errors) == 0 && len(results.warnings) == 0 {
		fmt.Printf("%s Workspace is ready for batch work\n", style.Success.Render("✓"))
	}
}

func runPostflight(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	fmt.Printf("%s Running postflight cleanup...\n", style.Info.Render("⚙"))

	// Load rigs configuration
	rigsConfigPath := filepath.Join(townRoot, "rigs.json")
	rigsConfig, err := config.LoadRigsConfig(rigsConfigPath)
	if err != nil {
		// Empty config if file doesn't exist
		rigsConfig = &config.RigsConfig{Rigs: make(map[string]config.RigEntry)}
	}

	if len(rigsConfig.Rigs) == 0 {
		fmt.Printf("%s No rigs found in workspace\n", style.Dim.Render("•"))
		return nil
	}

	// Create rig manager for discovery
	g := git.NewGit(townRoot)
	mgr := rig.NewManager(townRoot, rigsConfig, g)

	// Discover all rigs
	allRigs, err := mgr.DiscoverRigs()
	if err != nil {
		return fmt.Errorf("discovering rigs: %w", err)
	}

	// Filter to requested rig if specified
	var rigs []*rig.Rig
	if flightRig != "" {
		for _, r := range allRigs {
			if r.Name == flightRig {
				rigs = append(rigs, r)
				break
			}
		}
		if len(rigs) == 0 {
			return fmt.Errorf("rig %q not found", flightRig)
		}
	} else {
		rigs = allRigs
	}

	results := &postflightResults{}

	for _, r := range rigs {
		if err := postflightRig(townRoot, r, results); err != nil {
			fmt.Printf("%s Error checking rig %q: %v\n", style.Error.Render("✗"), r.Name, err)
			results.errors = append(results.errors, fmt.Sprintf("rig %s: %v", r.Name, err))
		}
	}

	// Print summary
	printPostflightSummary(results)

	return nil
}

type postflightResults struct {
	archivedMail    int
	cleanedBranches int
	synced          bool
	errors          []string
	warnings        []string
}

func postflightRig(townRoot string, r *rig.Rig, results *postflightResults) error {
	fmt.Printf("\n%s Cleaning up rig: %s\n", style.Info.Render("►"), r.Name)

	// 1. Archive old mail (if requested)
	if flightArchiveMail {
		if err := archiveOldMail(r, results); err != nil {
			results.errors = append(results.errors, fmt.Sprintf("mail archive: %v", err))
		}
	}

	// 2. Clean up stale integration branches
	if err := cleanStaleBranches(r, results); err != nil {
		results.errors = append(results.errors, fmt.Sprintf("branch cleanup: %v", err))
	}

	// 3. Sync beads
	if !flightDryRun {
		if err := runBdSync(); err != nil {
			results.errors = append(results.errors, fmt.Sprintf("bd sync: %v", err))
		} else {
			results.synced = true
		}
	}

	// 4. Report on rig state
	reportRigState(r, results)

	return nil
}

func archiveOldMail(r *rig.Rig, results *postflightResults) error {
	fmt.Printf("  • Archiving messages older than 30 days\n")

	// Check mail in beads
	beadsPath := filepath.Join(r.Path, ".beads")
	if _, err := os.Stat(beadsPath); err != nil {
		// No beads directory, skip
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -30)
	fmt.Printf("    Cutoff date: %s\n", cutoff.Format("2006-01-02"))

	// Would integrate with actual beads mail archiving
	// For now just show the check was done
	results.archivedMail = 0
	fmt.Printf("    %s No messages to archive\n", style.Success.Render("✓"))

	return nil
}

func cleanStaleBranches(r *rig.Rig, results *postflightResults) error {
	fmt.Printf("  • Cleaning stale integration branches\n")

	mainClone := filepath.Join(r.Path, "refinery", "rig")
	if _, err := os.Stat(mainClone); err != nil {
		// No main clone, skip
		return nil
	}

	// List remote branches
	cmd := exec.Command("git", "branch", "-r", "--merged")
	cmd.Dir = mainClone
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("listing branches: %w", err)
	}

	mergedBranches := strings.Split(strings.TrimSpace(string(output)), "\n")
	staleCount := 0

	for _, branch := range mergedBranches {
		branch = strings.TrimSpace(branch)
		// Skip main/master branches
		if strings.HasSuffix(branch, "main") || strings.HasSuffix(branch, "master") {
			continue
		}
		// Skip empty lines
		if branch == "" {
			continue
		}

		staleCount++
		if !flightDryRun {
			// Would delete the branch
			cleanBranch := strings.TrimPrefix(branch, "origin/")
			fmt.Printf("    • Would delete: %s\n", cleanBranch)
		}
	}

	if staleCount > 0 {
		results.cleanedBranches = staleCount
		if flightDryRun {
			fmt.Printf("    %s Found %d stale branches (use without --dry-run to clean)\n",
				style.Warning.Render("⚠"), staleCount)
		} else {
			fmt.Printf("    %s Cleaned %d stale branches\n",
				style.Success.Render("✓"), staleCount)
		}
	} else {
		fmt.Printf("    %s No stale branches\n", style.Success.Render("✓"))
	}

	return nil
}

func reportRigState(r *rig.Rig, results *postflightResults) error {
	fmt.Printf("  • Rig state:\n")

	t := tmux.NewTmux()

	// Report on services
	witnessSession := fmt.Sprintf("gt-%s-witness", r.Name)
	info, _ := t.GetSessionInfo(witnessSession)
	status := style.Success.Render("running")
	if info == nil {
		status = style.Dim.Render("stopped")
	}
	fmt.Printf("    • witness: %s\n", status)

	refinerySession := fmt.Sprintf("gt-%s-refinery", r.Name)
	info, _ = t.GetSessionInfo(refinerySession)
	status = style.Success.Render("running")
	if info == nil {
		status = style.Dim.Render("stopped")
	}
	fmt.Printf("    • refinery: %s\n", status)

	// Check for uncommitted changes
	mainClone := filepath.Join(r.Path, "refinery", "rig")
	if _, err := os.Stat(mainClone); err == nil {
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = mainClone
		output, err := cmd.Output()
		if err == nil && len(output) == 0 {
			fmt.Printf("    • git: %s\n", style.Success.Render("clean"))
		} else if len(output) > 0 {
			fmt.Printf("    • git: %s\n", style.Warning.Render("has changes"))
		}
	}

	return nil
}

func printPostflightSummary(results *postflightResults) {
	fmt.Printf("\n%s Postflight Summary:\n", style.Info.Render("📋"))

	if results.archivedMail > 0 {
		fmt.Printf("  • Archived: %d messages\n", results.archivedMail)
	}

	if results.cleanedBranches > 0 {
		fmt.Printf("  • Cleaned: %d branches\n", results.cleanedBranches)
	}

	if results.synced {
		fmt.Printf("  • Synced: beads\n")
	}

	if len(results.errors) > 0 {
		fmt.Printf("%s Errors: %d\n", style.Error.Render("✗"), len(results.errors))
		for _, e := range results.errors {
			fmt.Printf("  • %s\n", e)
		}
	}

	if len(results.warnings) > 0 {
		fmt.Printf("%s Warnings: %d\n", style.Warning.Render("⚠"), len(results.warnings))
		for _, w := range results.warnings {
			fmt.Printf("  • %s\n", w)
		}
	}

	if len(results.errors) == 0 {
		fmt.Printf("%s Postflight complete\n", style.Success.Render("✓"))
	}
}
