package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/style"
)

func runMQList(cmd *cobra.Command, args []string) error {
	rigName := args[0]

	_, r, _, err := getRefineryManager(rigName)
	if err != nil {
		return err
	}

	// Create beads wrapper for the rig - use BeadsPath() to get the git-synced location
	b := beads.New(r.BeadsPath())

	// Build list options - query for merge-request type
	// Priority -1 means no priority filter (otherwise 0 would filter to P0 only)
	opts := beads.ListOptions{
		Type:     "merge-request",
		Priority: -1,
	}

	// Apply status filter if specified
	if mqListStatus != "" {
		opts.Status = mqListStatus
	} else if !mqListReady {
		// Default to open if not showing ready
		opts.Status = "open"
	}

	var issues []*beads.Issue

	if mqListReady {
		// Use ready query which filters by no blockers
		allReady, err := b.Ready()
		if err != nil {
			return fmt.Errorf("querying ready MRs: %w", err)
		}
		// Filter to only merge-request type
		for _, issue := range allReady {
			if issue.Type == "merge-request" {
				issues = append(issues, issue)
			}
		}
	} else {
		issues, err = b.List(opts)
		if err != nil {
			return fmt.Errorf("querying merge queue: %w", err)
		}
	}

	// Apply additional filters and calculate scores
	type scoredIssue struct {
		issue *beads.Issue
		score float64
	}
	var scored []scoredIssue
	for _, issue := range issues {
		// Parse MR fields
		fields := beads.ParseMRFields(issue)

		// Filter by worker
		if mqListWorker != "" {
			worker := ""
			if fields != nil {
				worker = fields.Worker
			}
			if !strings.EqualFold(worker, mqListWorker) {
				continue
			}
		}

		// Filter by epic (target branch)
		if mqListEpic != "" {
			target := ""
			if fields != nil {
				target = fields.Target
			}
			expectedTarget := "integration/" + mqListEpic
			if target != expectedTarget {
				continue
			}
		}

		// Calculate score
		var retryCount int
		var convoyCreatedAt string
		if fields != nil {
			retryCount = fields.RetryCount
			// TODO: Look up convoy created_at from convoy bead if ConvoyID is set
			// For now, convoy age is 0 (no starvation prevention until convoy lookup is implemented)
			_ = fields.ConvoyID // Suppress unused warning
		}

		scoreInput := beads.MRScoreInput{
			Priority:        issue.Priority,
			CreatedAt:       issue.CreatedAt,
			ConvoyCreatedAt: convoyCreatedAt,
			RetryCount:      retryCount,
		}
		score := beads.ScoreMRIssueWithDefaults(scoreInput)

		scored = append(scored, scoredIssue{issue: issue, score: score})
	}

	// Sort by score descending (highest priority first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Extract filtered issues for JSON output
	filtered := make([]*beads.Issue, len(scored))
	for i, s := range scored {
		filtered[i] = s.issue
	}

	// JSON output
	if mqListJSON {
		return outputJSON(filtered)
	}

	// Human-readable output
	fmt.Printf("%s Merge queue for '%s':\n\n", style.Bold.Render("MQ"), rigName)

	if len(scored) == 0 {
		fmt.Printf("  %s\n", style.Dim.Render("(empty)"))
		return nil
	}

	// Create styled table with SCORE and CONVOY columns
	table := style.NewTable(
		style.Column{Name: "ID", Width: 12},
		style.Column{Name: "SCORE", Width: 7, Align: style.AlignRight},
		style.Column{Name: "PRI", Width: 4},
		style.Column{Name: "CONVOY", Width: 12},
		style.Column{Name: "BRANCH", Width: 26},
		style.Column{Name: "AGE", Width: 6, Align: style.AlignRight},
	)

	// Add rows
	for _, s := range scored {
		issue := s.issue
		fields := beads.ParseMRFields(issue)

		// Get convoy info
		convoy := "(none)"
		if fields != nil && fields.ConvoyID != "" {
			convoy = fields.ConvoyID
			if len(convoy) > 12 {
				convoy = convoy[:12]
			}
		}

		// Get MR fields
		branch := ""
		if fields != nil {
			branch = fields.Branch
		}

		// Format priority with color
		priority := fmt.Sprintf("P%d", issue.Priority)
		if issue.Priority <= 1 {
			priority = style.Error.Render(priority)
		} else if issue.Priority == 2 {
			priority = style.Warning.Render(priority)
		}

		// Calculate age
		age := formatMRAge(issue.CreatedAt)

		// Truncate ID if needed
		displayID := issue.ID
		if len(displayID) > 12 {
			displayID = displayID[:12]
		}

		// Format score
		scoreStr := fmt.Sprintf("%.1f", s.score)

		table.AddRow(displayID, scoreStr, priority, convoy, branch, style.Dim.Render(age))
	}

	fmt.Print(table.Render())

	// Show blocking details below table
	for _, s := range scored {
		issue := s.issue
		displayStatus := issue.Status
		if issue.Status == "open" && (len(issue.BlockedBy) > 0 || issue.BlockedByCount > 0) {
			displayStatus = "blocked"
		}
		if displayStatus == "blocked" && len(issue.BlockedBy) > 0 {
			displayID := issue.ID
			if len(displayID) > 12 {
				displayID = displayID[:12]
			}
			fmt.Printf("  %s %s\n", style.Dim.Render(displayID+":"),
				style.Dim.Render(fmt.Sprintf("waiting on %s", issue.BlockedBy[0])))
		}
	}

	return nil
}

// formatMRAge formats the age of an MR from its created_at timestamp.
func formatMRAge(createdAt string) string {
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		// Try other formats
		t, err = time.Parse("2006-01-02T15:04:05Z", createdAt)
		if err != nil {
			return "?"
		}
	}

	d := time.Since(t)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// outputJSON outputs data as JSON.
func outputJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
