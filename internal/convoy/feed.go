package convoy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FeedResult contains the result of a convoy feed operation.
type FeedResult struct {
	ConvoyID   string   `json:"convoy_id"`
	Dispatched []string `json:"dispatched"`          // Issues that were successfully dispatched
	Pending    []string `json:"pending,omitempty"`   // Issues still waiting (no capacity)
	Errors     []string `json:"errors,omitempty"`    // Errors during dispatch
}

// FeedOptions configures the feed operation.
type FeedOptions struct {
	DryRun     bool   // If true, don't actually dispatch
	MaxSling   int    // Maximum number of issues to sling (0 = unlimited)
	TargetRig  string // Specific rig to dispatch to (empty = auto-detect from issue prefix)
}

// Feed dispatches ready issues from a stranded convoy to available polecats.
// This is a single-pass operation: it feeds what it can and exits.
// The Deacon patrol loop handles repeated checking.
func Feed(townRoot string, convoyID string, opts FeedOptions) (*FeedResult, error) {
	townBeads := filepath.Join(townRoot, ".beads")

	// Get ready issues for this convoy
	readyIssues, err := getReadyIssuesForConvoy(townBeads, convoyID)
	if err != nil {
		return nil, fmt.Errorf("getting ready issues: %w", err)
	}

	if len(readyIssues) == 0 {
		return &FeedResult{
			ConvoyID:   convoyID,
			Dispatched: nil,
			Pending:    nil,
		}, nil
	}

	// Get idle polecat capacity across rigs
	idlePolecats, err := getIdlePolecats(townRoot)
	if err != nil {
		return nil, fmt.Errorf("getting idle polecats: %w", err)
	}

	result := &FeedResult{
		ConvoyID: convoyID,
	}

	// Dispatch issues to available polecats
	dispatched := 0
	for _, issue := range readyIssues {
		// Check max sling limit
		if opts.MaxSling > 0 && dispatched >= opts.MaxSling {
			result.Pending = append(result.Pending, issue.ID)
			continue
		}

		// Find a suitable polecat
		target := findTargetForIssue(issue, idlePolecats, opts.TargetRig)
		if target == "" {
			result.Pending = append(result.Pending, issue.ID)
			continue
		}

		if opts.DryRun {
			fmt.Printf("[DRY RUN] Would sling %s to %s\n", issue.ID, target)
			result.Dispatched = append(result.Dispatched, issue.ID)
			dispatched++
			continue
		}

		// Dispatch via gt sling
		if err := slingIssue(townRoot, issue.ID, target); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", issue.ID, err))
			continue
		}

		result.Dispatched = append(result.Dispatched, issue.ID)
		dispatched++

		// Mark this polecat as used (remove from available pool)
		for i, p := range idlePolecats {
			if p.Target == target {
				idlePolecats = append(idlePolecats[:i], idlePolecats[i+1:]...)
				break
			}
		}
	}

	return result, nil
}

// idlePolecat represents an available polecat for dispatch.
type idlePolecat struct {
	Rig    string
	Name   string
	Target string // Full target path: rig/polecats/name or just rig for auto-spawn
}

// getIdlePolecats finds polecats across all rigs that have no hooked work.
func getIdlePolecats(townRoot string) ([]idlePolecat, error) {
	// Discover rigs
	rigsPath := filepath.Join(townRoot, "mayor", "rigs.json")
	data, err := os.ReadFile(rigsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No rigs configured
		}
		return nil, err
	}

	var rigsConfig struct {
		Rigs map[string]interface{} `json:"rigs"`
	}
	if err := json.Unmarshal(data, &rigsConfig); err != nil {
		return nil, err
	}

	var idlePolecats []idlePolecat

	for rigName := range rigsConfig.Rigs {
		rigPath := filepath.Join(townRoot, rigName)
		polecatsDir := filepath.Join(rigPath, "polecats")

		entries, err := os.ReadDir(polecatsDir)
		if err != nil {
			continue // Rig might not have polecats dir
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			polecatName := entry.Name()
			assignee := fmt.Sprintf("%s/polecats/%s", rigName, polecatName)

			// Check if polecat has hooked work
			if !hasHookedWork(rigPath, assignee) {
				idlePolecats = append(idlePolecats, idlePolecat{
					Rig:    rigName,
					Name:   polecatName,
					Target: assignee,
				})
			}
		}
	}

	return idlePolecats, nil
}

// hasHookedWork checks if an agent has hooked work assigned.
func hasHookedWork(beadsDir, assignee string) bool {
	cmd := exec.Command("bd", "list", "--status=hooked", "--assignee="+assignee, "--json")
	cmd.Dir = beadsDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false // Assume no hooked work on error
	}

	var hooked []interface{}
	if err := json.Unmarshal(stdout.Bytes(), &hooked); err != nil {
		return false
	}

	return len(hooked) > 0
}

// findTargetForIssue selects a polecat to handle the issue.
// Prefers polecats from the same rig as the issue (based on prefix).
func findTargetForIssue(issue ReadyIssue, polecats []idlePolecat, targetRig string) string {
	if len(polecats) == 0 {
		return ""
	}

	// If specific rig requested, filter to that rig
	if targetRig != "" {
		for _, p := range polecats {
			if p.Rig == targetRig {
				return p.Target
			}
		}
		// No polecats in target rig - return empty to use rig for auto-spawn
		return ""
	}

	// Try to match issue prefix to rig
	// Issue ID format: prefix-rest (e.g., gt-abc123, bd-xyz456)
	issueRig := extractRigFromIssueID(issue.ID)
	if issueRig != "" {
		for _, p := range polecats {
			if p.Rig == issueRig {
				return p.Target
			}
		}
	}

	// No rig match - use first available polecat
	return polecats[0].Target
}

// extractRigFromIssueID attempts to determine the rig from an issue ID.
// Uses common prefix mappings (gt- -> gastown, bd- -> beads, etc.)
func extractRigFromIssueID(issueID string) string {
	parts := strings.SplitN(issueID, "-", 2)
	if len(parts) < 2 {
		return ""
	}

	prefix := strings.ToLower(parts[0])

	// Common prefix mappings
	// These should match the routes.jsonl prefixes
	prefixToRig := map[string]string{
		"gt": "gastown",
		"bd": "beads",
		"gp": "greenplace",
		"hq": "", // town-level, no specific rig
	}

	if rig, ok := prefixToRig[prefix]; ok {
		return rig
	}

	return ""
}

// slingIssue dispatches an issue to a target using gt sling.
func slingIssue(townRoot, issueID, target string) error {
	cmd := exec.Command("gt", "sling", issueID, target, "--no-convoy")
	cmd.Dir = townRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
