package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/style"
)

// MQ next command flags
var (
	mqNextStrategy string
	mqNextJSON     bool
)

var mqNextCmd = &cobra.Command{
	Use:   "next <rig>",
	Short: "Get the next MR to process from the queue",
	Long: `Return the highest-priority merge request from the queue.

By default, uses priority scoring which considers:
  - Issue priority (P0-P4)
  - Convoy age (prevents starvation of batch work)
  - Retry count (deprioritizes thrashing MRs)
  - MR age (FIFO tiebreaker)

Use --strategy=fifo to revert to simple first-in-first-out ordering.

Examples:
  gt mq next sai              # Highest-score MR
  gt mq next sai --strategy=fifo    # Oldest MR (legacy behavior)
  gt mq next sai --json             # JSON output for automation`,
	Args: cobra.ExactArgs(1),
	RunE: runMQNext,
}

func init() {
	mqNextCmd.Flags().StringVar(&mqNextStrategy, "strategy", "priority", "Selection strategy: priority (default) or fifo")
	mqNextCmd.Flags().BoolVar(&mqNextJSON, "json", false, "Output as JSON")

	mqCmd.AddCommand(mqNextCmd)
}

func runMQNext(cmd *cobra.Command, args []string) error {
	rigName := args[0]

	_, r, _, err := getRefineryManager(rigName)
	if err != nil {
		return err
	}

	// Create beads wrapper for the rig
	b := beads.New(r.BeadsPath())

	// Query for open merge-request issues
	opts := beads.ListOptions{
		Type:     "merge-request",
		Status:   "open",
		Priority: -1,
	}

	issues, err := b.List(opts)
	if err != nil {
		return fmt.Errorf("querying merge queue: %w", err)
	}

	// Filter to only ready MRs (no blockers)
	var ready []*beads.Issue
	for _, issue := range issues {
		if len(issue.BlockedBy) == 0 && issue.BlockedByCount == 0 {
			ready = append(ready, issue)
		}
	}

	if len(ready) == 0 {
		if mqNextJSON {
			fmt.Println("null")
		} else {
			fmt.Println("No ready MRs in queue")
		}
		return nil
	}

	var next *beads.Issue
	var nextScore float64

	if mqNextStrategy == "fifo" {
		// FIFO: sort by created_at ascending, pick first
		sort.Slice(ready, func(i, j int) bool {
			return ready[i].CreatedAt < ready[j].CreatedAt
		})
		next = ready[0]
	} else {
		// Priority scoring: calculate scores and pick highest
		type scoredIssue struct {
			issue *beads.Issue
			score float64
		}
		var scored []scoredIssue

		for _, issue := range ready {
			fields := beads.ParseMRFields(issue)
			var retryCount int
			var convoyCreatedAt string
			if fields != nil {
				retryCount = fields.RetryCount
				// TODO: Look up convoy created_at from convoy bead if ConvoyID is set
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

		// Sort by score descending
		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		next = scored[0].issue
		nextScore = scored[0].score
	}

	// Output
	if mqNextJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(next)
	}

	// Human-readable output
	fields := beads.ParseMRFields(next)
	fmt.Printf("%s Next MR to process:\n\n", style.Bold.Render(">>"))
	fmt.Printf("  ID:       %s\n", next.ID)
	if mqNextStrategy != "fifo" {
		fmt.Printf("  Score:    %.1f\n", nextScore)
	}
	fmt.Printf("  Priority: P%d\n", next.Priority)
	if fields != nil {
		if fields.Branch != "" {
			fmt.Printf("  Branch:   %s\n", fields.Branch)
		}
		if fields.Worker != "" {
			fmt.Printf("  Worker:   %s\n", fields.Worker)
		}
		if fields.ConvoyID != "" {
			fmt.Printf("  Convoy:   %s\n", fields.ConvoyID)
		}
		if fields.RetryCount > 0 {
			fmt.Printf("  Retries:  %d\n", fields.RetryCount)
		}
	}
	fmt.Printf("  Age:      %s\n", formatMRAge(next.CreatedAt))

	return nil
}
