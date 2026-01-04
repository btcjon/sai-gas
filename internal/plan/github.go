// Package plan provides utilities for parsing planning documents into beads epics.
package plan

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubIssue represents a GitHub issue.
type GitHubIssue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
	URL    string   `json:"url"`
}

// ghIssueRaw is the raw format from gh CLI.
type ghIssueRaw struct {
	Number int `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	URL string `json:"url"`
}

// FetchGitHubIssues fetches issues from a GitHub repository by label.
// Uses the gh CLI which must be installed and authenticated.
func FetchGitHubIssues(repo, label string) ([]GitHubIssue, error) {
	// Validate inputs
	if repo == "" {
		return nil, fmt.Errorf("repository is required (format: owner/repo)")
	}

	// Build gh command
	args := []string{
		"issue", "list",
		"--repo", repo,
		"--json", "number,title,body,state,labels,url",
		"--limit", "100",
	}

	if label != "" {
		args = append(args, "--label", label)
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh CLI error: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("running gh CLI: %w", err)
	}

	// Parse JSON output
	var rawIssues []ghIssueRaw
	if err := json.Unmarshal(output, &rawIssues); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}

	// Convert to our format
	issues := make([]GitHubIssue, len(rawIssues))
	for i, raw := range rawIssues {
		labels := make([]string, len(raw.Labels))
		for j, l := range raw.Labels {
			labels[j] = l.Name
		}

		issues[i] = GitHubIssue{
			Number: raw.Number,
			Title:  raw.Title,
			Body:   raw.Body,
			State:  raw.State,
			Labels: labels,
			URL:    raw.URL,
		}
	}

	return issues, nil
}

// GitHubIssuesToPlan converts a list of GitHub issues into a Plan.
// Each issue becomes a task. The epic title is derived from the label or repo name.
func GitHubIssuesToPlan(issues []GitHubIssue, epicTitle string) *Plan {
	plan := &Plan{
		Title:  epicTitle,
		Phases: make([]Phase, 0),
		Tasks:  make([]Task, 0),
	}

	// Create a single phase for all issues
	phase := Phase{
		Name:  "Imported Issues",
		Order: 1,
		Tasks: make([]Task, 0),
	}

	for i, issue := range issues {
		task := Task{
			Title:     fmt.Sprintf("#%d: %s", issue.Number, issue.Title),
			Done:      issue.State == "closed",
			Phase:     phase.Name,
			Order:     i + 1,
			DependsOn: nil,
			RawLine:   issue.URL,
		}

		phase.Tasks = append(phase.Tasks, task)
		plan.Tasks = append(plan.Tasks, task)
	}

	plan.Phases = append(plan.Phases, phase)
	return plan
}

// ParseRepoString parses "owner/repo" format and validates it.
func ParseRepoString(repo string) (owner, name string, err error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format: expected 'owner/repo', got %q", repo)
	}

	owner = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])

	if owner == "" || name == "" {
		return "", "", fmt.Errorf("invalid repo format: owner and repo name cannot be empty")
	}

	return owner, name, nil
}
