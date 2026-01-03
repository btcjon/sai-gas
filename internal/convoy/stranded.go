// Package convoy provides convoy management functionality.
package convoy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// StrandedConvoy represents a convoy that has ready work but no polecats working on it.
type StrandedConvoy struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	ReadyIssues []ReadyIssue  `json:"ready_issues"`
}

// ReadyIssue represents an issue that's ready to work (not blocked, no assignee).
type ReadyIssue struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	IssueType string `json:"issue_type,omitempty"`
}

// convoyInfo holds basic convoy data from bd.
type convoyInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// trackedIssueInfo holds data about issues tracked by a convoy.
type trackedIssueInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Assignee string `json:"assignee,omitempty"`
}

// DetectStranded finds all stranded convoys in the town.
// A convoy is stranded if:
//   - It is open
//   - It has tracked issues where:
//     - status = open (not in_progress, not closed)
//     - not blocked (dependencies met)
//     - no assignee (or assignee session is dead)
func DetectStranded(townRoot string) ([]StrandedConvoy, error) {
	townBeads := filepath.Join(townRoot, ".beads")

	// Get all open convoys
	convoys, err := listOpenConvoys(townBeads)
	if err != nil {
		return nil, fmt.Errorf("listing open convoys: %w", err)
	}

	var stranded []StrandedConvoy
	for _, c := range convoys {
		readyIssues, err := getReadyIssuesForConvoy(townBeads, c.ID)
		if err != nil {
			// Log but continue - one convoy error shouldn't stop detection
			continue
		}

		if len(readyIssues) > 0 {
			stranded = append(stranded, StrandedConvoy{
				ID:          c.ID,
				Title:       c.Title,
				ReadyIssues: readyIssues,
			})
		}
	}

	return stranded, nil
}

// listOpenConvoys returns all open convoys from town beads.
func listOpenConvoys(beadsDir string) ([]convoyInfo, error) {
	listArgs := []string{"list", "--type=convoy", "--status=open", "--json"}

	cmd := exec.Command("bd", listArgs...)
	cmd.Dir = beadsDir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("bd list: %w", err)
	}

	var convoys []convoyInfo
	if err := json.Unmarshal(stdout.Bytes(), &convoys); err != nil {
		return nil, fmt.Errorf("parsing convoys: %w", err)
	}

	return convoys, nil
}

// getReadyIssuesForConvoy returns issues tracked by the convoy that are ready to work.
// Ready = open status, not blocked, no assignee (or dead assignee session).
func getReadyIssuesForConvoy(beadsDir, convoyID string) ([]ReadyIssue, error) {
	dbPath := filepath.Join(beadsDir, "beads.db")

	// Query tracked dependencies from SQLite
	// Escape single quotes to prevent SQL injection
	safeConvoyID := strings.ReplaceAll(convoyID, "'", "''")
	queryCmd := exec.Command("sqlite3", "-json", dbPath,
		fmt.Sprintf(`SELECT depends_on_id FROM dependencies WHERE issue_id = '%s' AND type = 'tracks'`, safeConvoyID))

	var stdout bytes.Buffer
	queryCmd.Stdout = &stdout
	if err := queryCmd.Run(); err != nil {
		return nil, fmt.Errorf("querying dependencies: %w", err)
	}

	var deps []struct {
		DependsOnID string `json:"depends_on_id"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &deps); err != nil {
		// Empty result is valid
		if len(stdout.Bytes()) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing dependencies: %w", err)
	}

	// Collect issue IDs (normalize external refs)
	var issueIDs []string
	for _, dep := range deps {
		issueID := dep.DependsOnID

		// Handle external reference format: external:rig:issue-id
		if strings.HasPrefix(issueID, "external:") {
			parts := strings.SplitN(issueID, ":", 3)
			if len(parts) == 3 {
				issueID = parts[2]
			}
		}

		issueIDs = append(issueIDs, issueID)
	}

	if len(issueIDs) == 0 {
		return nil, nil
	}

	// Get details for all tracked issues
	var readyIssues []ReadyIssue
	for _, issueID := range issueIDs {
		info, err := getIssueInfo(issueID)
		if err != nil {
			continue // Skip issues we can't query
		}

		// Check if issue is ready to work:
		// 1. Status is "open" (not in_progress, not closed)
		// 2. Not blocked (dependencies met) - we check this separately
		// 3. No assignee
		if info.Status != "open" {
			continue
		}

		if info.Assignee != "" {
			// Has assignee - check if session is alive
			// For now, we assume assignee means not stranded
			// TODO: Check if assignee session is dead
			continue
		}

		// Check if blocked
		if isBlocked(issueID) {
			continue
		}

		readyIssues = append(readyIssues, ReadyIssue{
			ID:    issueID,
			Title: info.Title,
		})
	}

	return readyIssues, nil
}

// getIssueInfo fetches issue details by ID.
func getIssueInfo(issueID string) (*trackedIssueInfo, error) {
	showCmd := exec.Command("bd", "show", issueID, "--json")
	var stdout bytes.Buffer
	showCmd.Stdout = &stdout

	if err := showCmd.Run(); err != nil {
		return nil, err
	}

	var issues []trackedIssueInfo
	if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil || len(issues) == 0 {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	return &issues[0], nil
}

// isBlocked checks if an issue has unmet dependencies.
func isBlocked(issueID string) bool {
	// Use bd blocked to check if issue is blocked
	cmd := exec.Command("bd", "blocked", issueID)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Error running command - assume not blocked
		return false
	}

	// If output contains the issue ID, it's blocked
	return strings.Contains(stdout.String(), issueID)
}
