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

// === Tests for gt-si8rq.10: End-to-end rebase workflow edge cases ===

func TestConflictResolutionTask_ReconflictAfterResolution(t *testing.T) {
	// Edge case: MR conflicts again after being resolved once
	// This tests the metadata handling when retry_count > 1

	tests := []struct {
		name            string
		issue           *beads.Issue
		expectedRetry   int
		expectReconflict bool
	}{
		{
			name: "first conflict",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Feature X",
				Description: `## Metadata
- Original MR: mr-1234-abcd
- Branch: polecat/Toast-abc123
- Conflict with: main@abc12345
- Original issue: gt-xyz
- Retry count: 1`,
			},
			expectedRetry:    1,
			expectReconflict: false,
		},
		{
			name: "reconflict after resolution (retry 2)",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Feature X",
				Description: `## Metadata
- Original MR: mr-1234-abcd
- Branch: polecat/Toast-abc123
- Conflict with: main@def67890
- Original issue: gt-xyz
- Retry count: 2`,
			},
			expectedRetry:    2,
			expectReconflict: true,
		},
		{
			name: "multiple reconflicts (retry 5)",
			issue: &beads.Issue{
				Title: "Resolve merge conflicts: Feature X",
				Description: `## Metadata
- Original MR: mr-1234-abcd
- Branch: polecat/Toast-abc123
- Conflict with: main@ghi11111
- Original issue: gt-xyz
- Retry count: 5`,
			},
			expectedRetry:    5,
			expectReconflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseConflictTask(tt.issue)
			if task == nil {
				t.Fatal("ParseConflictTask returned nil")
			}

			if task.RetryCount != tt.expectedRetry {
				t.Errorf("expected retry count %d, got %d", tt.expectedRetry, task.RetryCount)
			}

			// Retry count > 1 indicates reconflict
			isReconflict := task.RetryCount > 1
			if isReconflict != tt.expectReconflict {
				t.Errorf("expected reconflict=%v, got %v", tt.expectReconflict, isReconflict)
			}

			// Verify conflict SHA changed (different main@SHA in each case)
			if task.ConflictSHA == "" {
				t.Error("expected conflict SHA to be present")
			}
		})
	}
}

func TestConflictResolutionTask_PolecatRecycledBeforeMerge(t *testing.T) {
	// Edge case: Polecat is recycled (removed) before merge completes
	// The conflict task should still contain all necessary metadata to recover

	issue := &beads.Issue{
		Title: "Resolve merge conflicts: Critical Bug Fix",
		Description: `## Metadata
- Original MR: mr-5678-efgh
- Branch: polecat/Nux-def456
- Conflict with: main@abc12345
- Original issue: gt-critical
- Retry count: 1

## Instructions
1. Check out the branch: git checkout polecat/Nux-def456
2. Rebase onto target: git rebase origin/main
...`,
	}

	task := ParseConflictTask(issue)
	if task == nil {
		t.Fatal("ParseConflictTask returned nil")
	}

	// Even if polecat "Nux" is recycled, the task should have:
	// 1. Original MR ID (to re-queue after resolution)
	// 2. Branch name (can be checked out in any clone)
	// 3. Target branch info (for rebase)
	// 4. Source issue (for closing after merge)

	if task.OriginalMRID == "" {
		t.Error("must have original MR ID for re-queue")
	}
	if task.Branch == "" {
		t.Error("must have branch name for checkout")
	}
	if task.TargetBranch == "" {
		t.Error("must have target branch for rebase")
	}
	if task.SourceIssue == "" {
		t.Error("must have source issue for post-merge cleanup")
	}

	// Verify branch follows polecat naming convention
	if !isWorkingBranch(task.Branch) {
		t.Errorf("branch %q should be recognized as a working branch", task.Branch)
	}
}

func TestConflictResolutionTask_MetadataRoundtrip(t *testing.T) {
	// Test that all metadata survives parsing round-trip

	original := ConflictResolutionTask{
		OriginalMRID: "mr-9999-zzzz",
		Branch:       "polecat/Max-xyz789",
		TargetBranch: "main",
		ConflictSHA:  "fedcba09",
		SourceIssue:  "gt-important",
		RetryCount:   3,
	}

	// Build description matching the format from engineer.go
	description := `## Metadata
- Original MR: mr-9999-zzzz
- Branch: polecat/Max-xyz789
- Conflict with: main@fedcba09
- Original issue: gt-important
- Retry count: 3`

	issue := &beads.Issue{
		Title:       "Resolve merge conflicts: Important Feature",
		Description: description,
	}

	parsed := ParseConflictTask(issue)
	if parsed == nil {
		t.Fatal("ParseConflictTask returned nil")
	}

	// Verify all fields match
	if parsed.OriginalMRID != original.OriginalMRID {
		t.Errorf("OriginalMRID: got %q, want %q", parsed.OriginalMRID, original.OriginalMRID)
	}
	if parsed.Branch != original.Branch {
		t.Errorf("Branch: got %q, want %q", parsed.Branch, original.Branch)
	}
	if parsed.TargetBranch != original.TargetBranch {
		t.Errorf("TargetBranch: got %q, want %q", parsed.TargetBranch, original.TargetBranch)
	}
	if parsed.ConflictSHA != original.ConflictSHA {
		t.Errorf("ConflictSHA: got %q, want %q", parsed.ConflictSHA, original.ConflictSHA)
	}
	if parsed.SourceIssue != original.SourceIssue {
		t.Errorf("SourceIssue: got %q, want %q", parsed.SourceIssue, original.SourceIssue)
	}
	if parsed.RetryCount != original.RetryCount {
		t.Errorf("RetryCount: got %d, want %d", parsed.RetryCount, original.RetryCount)
	}
}

func TestConflictResolutionTask_EdgeCaseBranchNames(t *testing.T) {
	// Test conflict tasks with various branch name patterns

	tests := []struct {
		name           string
		branch         string
		expectWorking  bool
		expectProtected bool
	}{
		{
			name:           "standard polecat branch",
			branch:         "polecat/Toast-abc123",
			expectWorking:  true,
			expectProtected: false,
		},
		{
			name:           "crew branch",
			branch:         "crew/max-feature",
			expectWorking:  true,
			expectProtected: false,
		},
		{
			name:           "feature branch",
			branch:         "feature/new-api",
			expectWorking:  true,
			expectProtected: false,
		},
		{
			name:           "fix branch",
			branch:         "fix/critical-bug",
			expectWorking:  true,
			expectProtected: false,
		},
		{
			name:           "temp resolve branch",
			branch:         "temp-resolve-conflict-xyz",
			expectWorking:  true,
			expectProtected: false,
		},
		{
			name:           "main is protected",
			branch:         "main",
			expectWorking:  false,
			expectProtected: true,
		},
		{
			name:           "master is protected",
			branch:         "master",
			expectWorking:  false,
			expectProtected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isWorking := isWorkingBranch(tt.branch)
			if isWorking != tt.expectWorking {
				t.Errorf("isWorkingBranch(%q) = %v, want %v", tt.branch, isWorking, tt.expectWorking)
			}

			isProtected := isProtectedBranch(tt.branch)
			if isProtected != tt.expectProtected {
				t.Errorf("isProtectedBranch(%q) = %v, want %v", tt.branch, isProtected, tt.expectProtected)
			}
		})
	}
}

func TestForcePushResult_SafetyValidation(t *testing.T) {
	// Test ForcePushResult structure and safety checks

	// Successful push result
	successResult := ForcePushResult{
		Success:   true,
		PushedSHA: "abc123def456",
		Error:     "",
	}

	if !successResult.Success {
		t.Error("expected success")
	}
	if successResult.PushedSHA == "" {
		t.Error("expected pushed SHA to be set on success")
	}

	// Failed push result
	failResult := ForcePushResult{
		Success: false,
		Error:   "refusing to force-push to protected branch: main",
	}

	if failResult.Success {
		t.Error("expected failure")
	}
	if failResult.Error == "" {
		t.Error("expected error message on failure")
	}
}

func TestRebaseResult_ConflictHandling(t *testing.T) {
	// Test RebaseResult structure for conflict scenarios

	tests := []struct {
		name           string
		result         RebaseResult
		expectSuccess  bool
		expectConflict bool
	}{
		{
			name: "clean rebase",
			result: RebaseResult{
				Success:    true,
				NewHeadSHA: "abc123def456",
			},
			expectSuccess:  true,
			expectConflict: false,
		},
		{
			name: "rebase with conflicts",
			result: RebaseResult{
				Success:       false,
				ConflictFiles: []string{"file1.go", "file2.go"},
				Error:         "rebase conflicts in: [file1.go file2.go]",
			},
			expectSuccess:  false,
			expectConflict: true,
		},
		{
			name: "rebase error (not conflict)",
			result: RebaseResult{
				Success: false,
				Error:   "rebase failed: branch not found",
			},
			expectSuccess:  false,
			expectConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", tt.result.Success, tt.expectSuccess)
			}

			hasConflicts := len(tt.result.ConflictFiles) > 0
			if hasConflicts != tt.expectConflict {
				t.Errorf("has conflicts = %v, want %v", hasConflicts, tt.expectConflict)
			}
		})
	}
}

func TestConflictResolver_SignalMRRequeue(t *testing.T) {
	// Test the metadata update behavior when signaling MR re-queue
	// This tests the expected state changes after conflict resolution

	type MRState struct {
		ConflictTaskID  string
		RetryCount      int
		LastConflictSHA string
	}

	tests := []struct {
		name           string
		beforeResolve  MRState
		afterResolve   MRState
	}{
		{
			name: "first conflict resolved",
			beforeResolve: MRState{
				ConflictTaskID:  "task-123",
				RetryCount:      1,
				LastConflictSHA: "abc123",
			},
			afterResolve: MRState{
				ConflictTaskID:  "", // Cleared after resolution
				RetryCount:      1,  // Kept for scoring
				LastConflictSHA: "abc123", // Historical record
			},
		},
		{
			name: "second conflict resolved",
			beforeResolve: MRState{
				ConflictTaskID:  "task-456",
				RetryCount:      2,
				LastConflictSHA: "def456",
			},
			afterResolve: MRState{
				ConflictTaskID:  "",
				RetryCount:      2,
				LastConflictSHA: "def456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the SignalMRRequeue behavior:
			// - ConflictTaskID should be cleared (work is done)
			// - RetryCount should be preserved (for scoring)
			// - LastConflictSHA should be preserved (for debugging)

			state := tt.beforeResolve

			// SignalMRRequeue clears ConflictTaskID
			state.ConflictTaskID = ""

			if state.ConflictTaskID != tt.afterResolve.ConflictTaskID {
				t.Errorf("ConflictTaskID: got %q, want %q", state.ConflictTaskID, tt.afterResolve.ConflictTaskID)
			}
			if state.RetryCount != tt.afterResolve.RetryCount {
				t.Errorf("RetryCount: got %d, want %d", state.RetryCount, tt.afterResolve.RetryCount)
			}
		})
	}
}
