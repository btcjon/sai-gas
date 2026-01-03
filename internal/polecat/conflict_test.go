// Package polecat provides polecat lifecycle management.
package polecat

import (
	"testing"

	"github.com/steveyegge/gastown/internal/beads"
)

func TestIsConflictResolutionTask(t *testing.T) {
	tests := []struct {
		name     string
		issue    *beads.Issue
		expected bool
	}{
		{
			name:     "nil issue",
			issue:    nil,
			expected: false,
		},
		{
			name: "regular task",
			issue: &beads.Issue{
				Title: "Implement feature X",
			},
			expected: false,
		},
		{
			name: "conflict resolution task",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Implement feature X",
			},
			expected: true,
		},
		{
			name: "similar but not conflict task",
			issue: &beads.Issue{
				Title: "Resolving merge conflicts discussion",
			},
			expected: false,
		},
		{
			name: "empty title",
			issue: &beads.Issue{
				Title: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsConflictResolutionTask(tt.issue)
			if got != tt.expected {
				t.Errorf("IsConflictResolutionTask() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseConflictTask(t *testing.T) {
	tests := []struct {
		name     string
		issue    *beads.Issue
		expected *ConflictResolutionTask
	}{
		{
			name: "not a conflict task",
			issue: &beads.Issue{
				Title:       "Regular task",
				Description: "Some description",
			},
			expected: nil,
		},
		{
			name: "conflict task with full metadata",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Implement feature X",
				Description: `Resolve merge conflicts for branch polecat/Toast-abc123

## Metadata
- Original MR: mr-1234-abcd
- Branch: polecat/Toast-abc123
- Conflict with: main@abc12345
- Original issue: gt-xyz
- Retry count: 2

## Instructions
1. Check out the branch...`,
			},
			expected: &ConflictResolutionTask{
				OriginalMRID: "mr-1234-abcd",
				Branch:       "polecat/Toast-abc123",
				TargetBranch: "main",
				ConflictSHA:  "abc12345",
				SourceIssue:  "gt-xyz",
				RetryCount:   2,
			},
		},
		{
			name: "conflict task with partial metadata",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Fix bug",
				Description: `## Metadata
- Original MR: mr-5678-efgh
- Branch: polecat/Nux-def456`,
			},
			expected: &ConflictResolutionTask{
				OriginalMRID: "mr-5678-efgh",
				Branch:       "polecat/Nux-def456",
				TargetBranch: "",
				ConflictSHA:  "",
				SourceIssue:  "",
				RetryCount:   0,
			},
		},
		{
			name: "conflict task with malformed conflict line",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Update docs",
				Description: `## Metadata
- Original MR: mr-9999
- Branch: feature/docs
- Conflict with: develop`,
			},
			expected: &ConflictResolutionTask{
				OriginalMRID: "mr-9999",
				Branch:       "feature/docs",
				TargetBranch: "develop",
				ConflictSHA:  "",
				SourceIssue:  "",
				RetryCount:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseConflictTask(tt.issue)

			if tt.expected == nil {
				if got != nil {
					t.Errorf("ParseConflictTask() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("ParseConflictTask() = nil, want %v", tt.expected)
				return
			}

			if got.OriginalMRID != tt.expected.OriginalMRID {
				t.Errorf("OriginalMRID = %q, want %q", got.OriginalMRID, tt.expected.OriginalMRID)
			}
			if got.Branch != tt.expected.Branch {
				t.Errorf("Branch = %q, want %q", got.Branch, tt.expected.Branch)
			}
			if got.TargetBranch != tt.expected.TargetBranch {
				t.Errorf("TargetBranch = %q, want %q", got.TargetBranch, tt.expected.TargetBranch)
			}
			if got.ConflictSHA != tt.expected.ConflictSHA {
				t.Errorf("ConflictSHA = %q, want %q", got.ConflictSHA, tt.expected.ConflictSHA)
			}
			if got.SourceIssue != tt.expected.SourceIssue {
				t.Errorf("SourceIssue = %q, want %q", got.SourceIssue, tt.expected.SourceIssue)
			}
			if got.RetryCount != tt.expected.RetryCount {
				t.Errorf("RetryCount = %d, want %d", got.RetryCount, tt.expected.RetryCount)
			}
		})
	}
}

func TestExtractBranchName(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"origin/polecat/Toast", "polecat/Toast"},
		{"polecat/Toast", "polecat/Toast"},
		{"origin/main", "main"},
		{"main", "main"},
		{"origin/feature/cool-feature", "feature/cool-feature"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			got := extractBranchName(tt.ref)
			if got != tt.expected {
				t.Errorf("extractBranchName(%q) = %q, want %q", tt.ref, got, tt.expected)
			}
		})
	}
}

func TestIsWorkingBranch(t *testing.T) {
	tests := []struct {
		branch   string
		expected bool
	}{
		{"polecat/Toast-abc123", true},
		{"polecat/Nux", true},
		{"crew/max", true},
		{"feature/new-feature", true},
		{"fix/bug-123", true},
		{"temp-resolve-xyz", true},
		{"main", false},
		{"master", false},
		{"develop", false},
		{"release", false},
		{"hotfix/urgent", false},
		{"some-random-branch", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := isWorkingBranch(tt.branch)
			if got != tt.expected {
				t.Errorf("isWorkingBranch(%q) = %v, want %v", tt.branch, got, tt.expected)
			}
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	tests := []struct {
		branch   string
		expected bool
	}{
		{"main", true},
		{"master", true},
		{"develop", true},
		{"release", true},
		{"origin/main", true},
		{"origin/master", true},
		{"polecat/Toast", false},
		{"feature/new", false},
		{"fix/bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := isProtectedBranch(tt.branch)
			if got != tt.expected {
				t.Errorf("isProtectedBranch(%q) = %v, want %v", tt.branch, got, tt.expected)
			}
		})
	}
}
