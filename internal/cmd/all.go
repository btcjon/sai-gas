package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/session"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
)

// All command flags
var (
	allStartAwakeOnly bool
	allStopForce      bool
	allStatusJSON     bool
	allRunOutput      bool
)

var allCmd = &cobra.Command{
	Use:     "all",
	GroupID: GroupAgents,
	Short:   "Batch operations for multiple polecats",
	Long: `Perform batch operations across multiple polecats.

SPEC PATTERNS:
  Toast              - Specific polecat (requires unique name)
  gastown/Toast      - rig/polecat format
  gastown/*          - All polecats in a rig
  *                  - All polecats everywhere

Examples:
  gt all start gastown/*             # Start all polecats in gastown rig
  gt all stop --force *              # Force-stop all polecats
  gt all status greenplace/* --json  # JSON status for greenplace polecats
  gt all run "echo hello" *          # Run command in all polecat sessions`,
	RunE: requireSubcommand,
}

var allStartCmd = &cobra.Command{
	Use:   "start [--awake-only] [specs...]",
	Short: "Start multiple polecat sessions",
	Long: `Start sessions for multiple polecats matching the specs.

Use --awake-only to skip polecats that are not in 'working' state.

Examples:
  gt all start gastown/*
  gt all start --awake-only gastown/* greenplace/*
  gt all start gastown/Toast greenplace/Furiosa`,
	RunE: runAllStart,
}

var allStopCmd = &cobra.Command{
	Use:   "stop [--force] [specs...]",
	Short: "Stop multiple polecat sessions",
	Long: `Stop sessions for multiple polecats matching the specs.

By default, gracefully stops sessions. Use --force to kill immediately.

Examples:
  gt all stop gastown/*
  gt all stop --force *
  gt all stop gastown/Toast greenplace/Furiosa`,
	RunE: runAllStop,
}

var allStatusCmd = &cobra.Command{
	Use:   "status [--json] [specs...]",
	Short: "Show status of multiple polecats",
	Long: `Show status for multiple polecats matching the specs.

If no specs provided, shows all polecats.

Examples:
  gt all status
  gt all status gastown/*
  gt all status * --json`,
	RunE: runAllStatus,
}

var allRunCmd = &cobra.Command{
	Use:   "run <command> [specs...]",
	Short: "Run command in multiple polecat sessions",
	Long: `Run a command in the tmux sessions of multiple polecats.

The command is sent to each session via tmux send-keys.

Examples:
  gt all run "cd src && ls" gastown/*
  gt all run "echo hello" *`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAllRun,
}

func init() {
	// Start flags
	allStartCmd.Flags().BoolVar(&allStartAwakeOnly, "awake-only", false, "Only start polecats in 'working' state")

	// Stop flags
	allStopCmd.Flags().BoolVarP(&allStopForce, "force", "f", false, "Force-kill sessions")

	// Status flags
	allStatusCmd.Flags().BoolVar(&allStatusJSON, "json", false, "Output as JSON")

	// Run flags
	allRunCmd.Flags().BoolVar(&allRunOutput, "output", false, "Show command output (requires attached session)")

	// Add subcommands
	allCmd.AddCommand(allStartCmd)
	allCmd.AddCommand(allStopCmd)
	allCmd.AddCommand(allStatusCmd)
	allCmd.AddCommand(allRunCmd)

	rootCmd.AddCommand(allCmd)
}

// PolecatSpec represents a single polecat with its rig.
type PolecatSpec struct {
	Rig  string
	Name string
}

// String returns the rig/name format.
func (p PolecatSpec) String() string {
	return fmt.Sprintf("%s/%s", p.Rig, p.Name)
}

// expandSpecs expands spec patterns to a list of polecats.
// Patterns:
//   - "Toast" - find polecat with name Toast (unique across rigs)
//   - "gastown/Toast" - specific rig/polecat
//   - "gastown/*" - all polecats in rig
//   - "*" - all polecats everywhere
func expandSpecs(specs []string) ([]PolecatSpec, error) {
	if len(specs) == 0 {
		specs = []string{"*"}
	}

	// Collect all available polecats from all rigs
	allRigs, _, err := getAllRigs()
	if err != nil {
		return nil, fmt.Errorf("getting rigs: %w", err)
	}

	type availablePolecat struct {
		rig  *rig.Rig
		name string
	}

	var available []availablePolecat
	for _, r := range allRigs {
		mgr, _, err := getPolecatManager(r.Name)
		if err != nil {
			continue // skip rig if we can't read it
		}

		polecats, err := mgr.List()
		if err != nil {
			continue // skip rig if we can't list polecats
		}

		for _, p := range polecats {
			available = append(available, availablePolecat{
				rig:  r,
				name: p.Name,
			})
		}
	}

	var result []PolecatSpec
	seenMap := make(map[string]bool)

	// Process each spec
	for _, spec := range specs {
		// "*" - all polecats
		if spec == "*" {
			for _, p := range available {
				key := p.rig.Name + "/" + p.name
				if !seenMap[key] {
					result = append(result, PolecatSpec{Rig: p.rig.Name, Name: p.name})
					seenMap[key] = true
				}
			}
			continue
		}

		// "rig/*" - all in rig
		if strings.HasSuffix(spec, "/*") {
			rigName := spec[:len(spec)-2]
			for _, p := range available {
				if p.rig.Name == rigName {
					key := p.rig.Name + "/" + p.name
					if !seenMap[key] {
						result = append(result, PolecatSpec{Rig: p.rig.Name, Name: p.name})
						seenMap[key] = true
					}
				}
			}
			continue
		}

		// "rig/name" - specific polecat
		if strings.Contains(spec, "/") {
			parts := strings.SplitN(spec, "/", 2)
			if len(parts) == 2 {
				rigName, polecatName := parts[0], parts[1]
				key := rigName + "/" + polecatName
				if !seenMap[key] {
					result = append(result, PolecatSpec{Rig: rigName, Name: polecatName})
					seenMap[key] = true
				}
				continue
			}
		}

		// "name" - find by name (must be unique)
		matches := 0
		var matchRig string
		for _, p := range available {
			if p.name == spec {
				matches++
				matchRig = p.rig.Name
			}
		}

		if matches == 0 {
			return nil, fmt.Errorf("polecat '%s' not found", spec)
		} else if matches > 1 {
			return nil, fmt.Errorf("polecat name '%s' is ambiguous (found in %d rigs, use 'rig/%s' format)", spec, matches, spec)
		}

		key := matchRig + "/" + spec
		if !seenMap[key] {
			result = append(result, PolecatSpec{Rig: matchRig, Name: spec})
			seenMap[key] = true
		}
	}

	// Sort for consistent output
	sort.Slice(result, func(i, j int) bool {
		if result[i].Rig != result[j].Rig {
			return result[i].Rig < result[j].Rig
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// runForAll executes a function for each polecat, collecting errors.
// Returns the count of successful operations and any errors.
func runForAll(polecats []PolecatSpec, action func(PolecatSpec) error) (int, []string) {
	var mu sync.Mutex
	var errors []string
	var wg sync.WaitGroup
	var successCount int

	// Use a semaphore to limit parallelism (avoid overwhelming system)
	semaphore := make(chan struct{}, 4) // 4 concurrent operations

	for _, p := range polecats {
		wg.Add(1)
		go func(spec PolecatSpec) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := action(spec); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", spec, err))
				mu.Unlock()
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(p)
	}

	wg.Wait()
	return successCount, errors
}

func runAllStart(cmd *cobra.Command, args []string) error {
	// Expand specs
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats found matching specs.")
		return nil
	}

	fmt.Printf("Starting %d polecat session(s)...\n\n", len(specs))

	t := tmux.NewTmux()
	successCount, errs := runForAll(specs, func(spec PolecatSpec) error {
		mgr, r, err := getPolecatManager(spec.Rig)
		if err != nil {
			return err
		}

		// Check if already running
		sessMgr := session.NewManager(t, r)
		running, _ := sessMgr.IsRunning(spec.Name)
		if running {
			fmt.Printf("  %s %s (already running)\n", style.Dim.Render("○"), spec)
			return nil
		}

		// Check state if --awake-only
		if allStartAwakeOnly {
			p, err := mgr.Get(spec.Name)
			if err != nil {
				return err
			}
			if p.State != polecat.StateWorking {
				fmt.Printf("  %s %s (skipped: state=%s, not 'working')\n", style.Dim.Render("○"), spec, p.State)
				return nil
			}
		}

		// Start session
		fmt.Printf("  %s %s...\n", style.Info.Render("●"), spec)
		if err := sessMgr.Start(spec.Name, session.StartOptions{}); err != nil {
			return err
		}

		return nil
	})

	fmt.Printf("\n%s Started %d session(s).\n", style.SuccessPrefix, successCount)

	if len(errs) > 0 {
		fmt.Printf("%s %d error(s):\n", style.Warning.Render("Warning:"), len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d start(s) failed", len(errs))
	}

	return nil
}

func runAllStop(cmd *cobra.Command, args []string) error {
	// Expand specs
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats found matching specs.")
		return nil
	}

	fmt.Printf("Stopping %d polecat session(s)...\n\n", len(specs))

	t := tmux.NewTmux()
	successCount, errs := runForAll(specs, func(spec PolecatSpec) error {
		_, r, err := getPolecatManager(spec.Rig)
		if err != nil {
			return err
		}

		// Check if running
		sessMgr := session.NewManager(t, r)
		running, _ := sessMgr.IsRunning(spec.Name)
		if !running {
			fmt.Printf("  %s %s (not running)\n", style.Dim.Render("○"), spec)
			return nil
		}

		// Stop session
		forceStr := ""
		if allStopForce {
			forceStr = " (forced)"
		}
		fmt.Printf("  %s %s%s...\n", style.Warning.Render("●"), spec, forceStr)
		if err := sessMgr.Stop(spec.Name, allStopForce); err != nil {
			return err
		}

		return nil
	})

	fmt.Printf("\n%s Stopped %d session(s).\n", style.SuccessPrefix, successCount)

	if len(errs) > 0 {
		fmt.Printf("%s %d error(s):\n", style.Warning.Render("Warning:"), len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d stop(s) failed", len(errs))
	}

	return nil
}

func runAllStatus(cmd *cobra.Command, args []string) error {
	// Expand specs
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats found.")
		return nil
	}

	// Collect status for all polecats
	type statusResult struct {
		Spec       PolecatSpec
		Status     *PolecatStatus
		Err        error
		Polecat    *polecat.Polecat
	}

	var mu sync.Mutex
	var results []statusResult
	var wg sync.WaitGroup

	t := tmux.NewTmux()

	for _, spec := range specs {
		wg.Add(1)
		go func(s PolecatSpec) {
			defer wg.Done()

			mgr, r, err := getPolecatManager(s.Rig)
			if err != nil {
				mu.Lock()
				results = append(results, statusResult{Spec: s, Err: err})
				mu.Unlock()
				return
			}

			p, err := mgr.Get(s.Name)
			if err != nil {
				mu.Lock()
				results = append(results, statusResult{Spec: s, Err: err})
				mu.Unlock()
				return
			}

			sessMgr := session.NewManager(t, r)
			sessInfo, _ := sessMgr.Status(s.Name)

			status := &PolecatStatus{
				Rig:            s.Rig,
				Name:           s.Name,
				State:          p.State,
				Issue:          p.Issue,
				ClonePath:      p.ClonePath,
				Branch:         p.Branch,
				SessionRunning: sessInfo.Running,
				SessionID:      sessInfo.SessionID,
				Attached:       sessInfo.Attached,
				Windows:        sessInfo.Windows,
			}
			if !sessInfo.Created.IsZero() {
				status.CreatedAt = sessInfo.Created.Format("2006-01-02 15:04:05")
			}
			if !sessInfo.LastActivity.IsZero() {
				status.LastActivity = sessInfo.LastActivity.Format("2006-01-02 15:04:05")
			}

			mu.Lock()
			results = append(results, statusResult{
				Spec:     s,
				Status:   status,
				Polecat:  p,
			})
			mu.Unlock()
		}(spec)
	}

	wg.Wait()

	// Sort results by rig/name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Spec.Rig != results[j].Spec.Rig {
			return results[i].Spec.Rig < results[j].Spec.Rig
		}
		return results[i].Spec.Name < results[j].Spec.Name
	})

	// JSON output
	if allStatusJSON {
		var jsonResults []interface{}
		for _, r := range results {
			if r.Err != nil {
				jsonResults = append(jsonResults, map[string]string{
					"rig":   r.Spec.Rig,
					"name":  r.Spec.Name,
					"error": r.Err.Error(),
				})
			} else {
				jsonResults = append(jsonResults, r.Status)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonResults)
	}

	// Human-readable output
	fmt.Printf("%s\n\n", style.Bold.Render("Polecat Status"))

	var errCount int
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  %s %s: %v\n", style.Error.Render("✗"), r.Spec, r.Err)
			errCount++
			continue
		}

		status := r.Status

		// Session indicator
		sessionStatus := style.Dim.Render("○")
		if status.SessionRunning {
			sessionStatus = style.Success.Render("●")
		}

		// State color
		stateStr := string(status.State)
		switch status.State {
		case polecat.StateWorking:
			stateStr = style.Info.Render(stateStr)
		case polecat.StateStuck:
			stateStr = style.Warning.Render(stateStr)
		case polecat.StateDone:
			stateStr = style.Success.Render(stateStr)
		default:
			stateStr = style.Dim.Render(stateStr)
		}

		fmt.Printf("  %s %s  %s\n", sessionStatus, style.Bold.Render(status.Rig+"/"+status.Name), stateStr)
		if status.Issue != "" {
			fmt.Printf("      Issue: %s\n", style.Dim.Render(status.Issue))
		}
	}

	if errCount > 0 {
		fmt.Printf("\n%s %d error(s) while collecting status\n", style.Warning.Render("Warning:"), errCount)
		return fmt.Errorf("%d status(es) failed", errCount)
	}

	return nil
}

func runAllRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("command required")
	}

	cmdStr := args[0]
	specs, err := expandSpecs(args[1:])
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats found matching specs.")
		return nil
	}

	fmt.Printf("Running command in %d session(s):\n", len(specs))
	fmt.Printf("  Command: %s\n\n", style.Dim.Render(cmdStr))

	t := tmux.NewTmux()
	successCount, errs := runForAll(specs, func(spec PolecatSpec) error {
		_, r, err := getPolecatManager(spec.Rig)
		if err != nil {
			return err
		}

		// Check if running
		sessMgr := session.NewManager(t, r)
		running, _ := sessMgr.IsRunning(spec.Name)
		if !running {
			fmt.Printf("  %s %s (session not running)\n", style.Dim.Render("○"), spec)
			return nil
		}

		// Send command to session
		sessionName := sessMgr.SessionName(spec.Name)
		fmt.Printf("  %s %s...\n", style.Info.Render("●"), spec)

		// Send keys: command + Enter
		if err := t.SendKeys(sessionName, cmdStr); err != nil {
			return err
		}

		return nil
	})

	fmt.Printf("\n%s Sent command to %d session(s).\n", style.SuccessPrefix, successCount)

	if len(errs) > 0 {
		fmt.Printf("%s %d error(s):\n", style.Warning.Render("Warning:"), len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d command(s) failed", len(errs))
	}

	return nil
}
