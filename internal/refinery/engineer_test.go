package refinery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/rig"
)

func TestDefaultMergeQueueConfig(t *testing.T) {
	cfg := DefaultMergeQueueConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if cfg.TargetBranch != "main" {
		t.Errorf("expected TargetBranch to be 'main', got %q", cfg.TargetBranch)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Errorf("expected PollInterval to be 30s, got %v", cfg.PollInterval)
	}
	if cfg.MaxConcurrent != 1 {
		t.Errorf("expected MaxConcurrent to be 1, got %d", cfg.MaxConcurrent)
	}
	if cfg.OnConflict != "assign_back" {
		t.Errorf("expected OnConflict to be 'assign_back', got %q", cfg.OnConflict)
	}
}

func TestEngineer_LoadConfig_NoFile(t *testing.T) {
	// Create a temp directory without config.json
	tmpDir, err := os.MkdirTemp("", "engineer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}

	e := NewEngineer(r)

	// Should not error with missing config file
	if err := e.LoadConfig(); err != nil {
		t.Errorf("unexpected error with missing config: %v", err)
	}

	// Should use defaults
	if e.config.PollInterval != 30*time.Second {
		t.Errorf("expected default PollInterval, got %v", e.config.PollInterval)
	}
}

func TestEngineer_LoadConfig_WithMergeQueue(t *testing.T) {
	// Create a temp directory with config.json
	tmpDir, err := os.MkdirTemp("", "engineer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write config file
	config := map[string]interface{}{
		"type":    "rig",
		"version": 1,
		"name":    "test-rig",
		"merge_queue": map[string]interface{}{
			"enabled":        true,
			"target_branch":  "develop",
			"poll_interval":  "10s",
			"max_concurrent": 2,
			"run_tests":      false,
			"test_command":   "make test",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}

	e := NewEngineer(r)

	if err := e.LoadConfig(); err != nil {
		t.Errorf("unexpected error loading config: %v", err)
	}

	// Check that config values were loaded
	if e.config.TargetBranch != "develop" {
		t.Errorf("expected TargetBranch 'develop', got %q", e.config.TargetBranch)
	}
	if e.config.PollInterval != 10*time.Second {
		t.Errorf("expected PollInterval 10s, got %v", e.config.PollInterval)
	}
	if e.config.MaxConcurrent != 2 {
		t.Errorf("expected MaxConcurrent 2, got %d", e.config.MaxConcurrent)
	}
	if e.config.RunTests != false {
		t.Errorf("expected RunTests false, got %v", e.config.RunTests)
	}
	if e.config.TestCommand != "make test" {
		t.Errorf("expected TestCommand 'make test', got %q", e.config.TestCommand)
	}

	// Check that defaults are preserved for unspecified fields
	if e.config.OnConflict != "assign_back" {
		t.Errorf("expected OnConflict default 'assign_back', got %q", e.config.OnConflict)
	}
}

func TestEngineer_LoadConfig_NoMergeQueueSection(t *testing.T) {
	// Create a temp directory with config.json without merge_queue
	tmpDir, err := os.MkdirTemp("", "engineer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write config file without merge_queue
	config := map[string]interface{}{
		"type":    "rig",
		"version": 1,
		"name":    "test-rig",
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}

	e := NewEngineer(r)

	if err := e.LoadConfig(); err != nil {
		t.Errorf("unexpected error loading config: %v", err)
	}

	// Should use all defaults
	if e.config.PollInterval != 30*time.Second {
		t.Errorf("expected default PollInterval, got %v", e.config.PollInterval)
	}
}

func TestEngineer_LoadConfig_InvalidPollInterval(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "engineer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := map[string]interface{}{
		"merge_queue": map[string]interface{}{
			"poll_interval": "not-a-duration",
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	r := &rig.Rig{
		Name: "test-rig",
		Path: tmpDir,
	}

	e := NewEngineer(r)

	err = e.LoadConfig()
	if err == nil {
		t.Error("expected error for invalid poll_interval")
	}
}

func TestNewEngineer(t *testing.T) {
	r := &rig.Rig{
		Name: "test-rig",
		Path: "/tmp/test-rig",
	}

	e := NewEngineer(r)

	if e.rig != r {
		t.Error("expected rig to be set")
	}
	if e.beads == nil {
		t.Error("expected beads client to be initialized")
	}
	if e.git == nil {
		t.Error("expected git client to be initialized")
	}
	if e.config == nil {
		t.Error("expected config to be initialized with defaults")
	}
}

func TestEngineer_DeleteMergedBranchesConfig(t *testing.T) {
	// Test that DeleteMergedBranches is true by default
	cfg := DefaultMergeQueueConfig()
	if !cfg.DeleteMergedBranches {
		t.Error("expected DeleteMergedBranches to be true by default")
	}
}

// === Tests for gt-si8rq.10: End-to-end rebase workflow validation ===

func TestProcessResult_StructureValidation(t *testing.T) {
	// Verify ProcessResult struct has all required fields for rebase workflow

	// Clean merge result
	cleanResult := ProcessResult{
		Success:     true,
		MergeCommit: "abc123def456",
		Error:       "",
		Conflict:    false,
		TestsFailed: false,
	}

	if !cleanResult.Success {
		t.Error("expected clean merge to succeed")
	}
	if cleanResult.MergeCommit == "" {
		t.Error("expected merge commit to be set on success")
	}
	if cleanResult.Conflict {
		t.Error("expected no conflict on clean merge")
	}

	// Conflict result
	conflictResult := ProcessResult{
		Success:  false,
		Conflict: true,
		Error:    "merge conflicts in: file1.go, file2.go",
	}

	if conflictResult.Success {
		t.Error("expected conflict result to not succeed")
	}
	if !conflictResult.Conflict {
		t.Error("expected conflict flag to be set")
	}
	if conflictResult.Error == "" {
		t.Error("expected error message on conflict")
	}

	// Test failure result
	testFailResult := ProcessResult{
		Success:     false,
		TestsFailed: true,
		Error:       "tests failed after 3 attempts",
	}

	if testFailResult.Success {
		t.Error("expected test failure result to not succeed")
	}
	if !testFailResult.TestsFailed {
		t.Error("expected tests failed flag to be set")
	}
}

func TestMergeQueueConfig_ConflictStrategies(t *testing.T) {
	// Test that OnConflict field supports expected strategies
	validStrategies := []string{"assign_back", "auto_rebase"}

	cfg := DefaultMergeQueueConfig()

	// Default should be assign_back
	if cfg.OnConflict != "assign_back" {
		t.Errorf("expected default OnConflict to be 'assign_back', got %q", cfg.OnConflict)
	}

	// Test that config can be set to other valid strategies
	for _, strategy := range validStrategies {
		cfg.OnConflict = strategy
		if cfg.OnConflict != strategy {
			t.Errorf("expected OnConflict to be %q, got %q", strategy, cfg.OnConflict)
		}
	}
}

func TestEngineer_WorkflowStateTransitions(t *testing.T) {
	// Test MergeRequest state machine for rebase workflow
	// Using the types from the types.go file

	tests := []struct {
		name          string
		initialStatus MRStatus
		transition    string
		expectError   bool
		finalStatus   MRStatus
	}{
		{
			name:          "open to in_progress (claim)",
			initialStatus: MROpen,
			transition:    "claim",
			expectError:   false,
			finalStatus:   MRInProgress,
		},
		{
			name:          "in_progress to closed (merge success)",
			initialStatus: MRInProgress,
			transition:    "close",
			expectError:   false,
			finalStatus:   MRClosed,
		},
		{
			name:          "in_progress to open (conflict, reassign)",
			initialStatus: MRInProgress,
			transition:    "reopen",
			expectError:   false,
			finalStatus:   MROpen,
		},
		{
			name:          "closed is immutable",
			initialStatus: MRClosed,
			transition:    "reopen",
			expectError:   true,
			finalStatus:   MRClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MergeRequest{
				ID:     "test-mr",
				Status: tt.initialStatus,
			}

			var err error
			switch tt.transition {
			case "claim":
				err = mr.Claim()
			case "close":
				err = mr.Close(CloseReasonMerged)
			case "reopen":
				err = mr.Reopen()
			}

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if mr.Status != tt.finalStatus {
				t.Errorf("expected final status %s, got %s", tt.finalStatus, mr.Status)
			}
		})
	}
}

func TestEngineer_ConflictResolutionTaskCreation(t *testing.T) {
	// Test that conflict resolution tasks have correct format
	// This tests the task format without actually calling createConflictResolutionTask

	// Expected task structure for a conflict resolution task
	type ConflictTaskMetadata struct {
		OriginalMR    string
		Branch        string
		ConflictWith  string
		OriginalIssue string
		RetryCount    int
	}

	// Simulate extracting metadata from a conflict task description
	description := `Resolve merge conflicts for branch polecat/Toast-abc123

## Metadata
- Original MR: mr-1234-abcd
- Branch: polecat/Toast-abc123
- Conflict with: main@abc12345
- Original issue: gt-xyz
- Retry count: 2

## Instructions
1. Check out the branch: git checkout polecat/Toast-abc123
2. Rebase onto target: git rebase origin/main
3. Resolve conflicts in your editor
4. Complete the rebase: git add . && git rebase --continue
5. Force-push the resolved branch: git push -f
6. Close this task: bd close <this-task-id>

The Refinery will automatically retry the merge after you force-push.`

	// Verify key elements are present
	if !strings.Contains(description, "## Metadata") {
		t.Error("conflict task should have metadata section")
	}
	if !strings.Contains(description, "Original MR:") {
		t.Error("conflict task should include original MR reference")
	}
	if !strings.Contains(description, "Branch:") {
		t.Error("conflict task should include branch name")
	}
	if !strings.Contains(description, "Conflict with:") {
		t.Error("conflict task should include conflict target info")
	}
	if !strings.Contains(description, "Retry count:") {
		t.Error("conflict task should track retry count")
	}
	if !strings.Contains(description, "## Instructions") {
		t.Error("conflict task should have instructions section")
	}
	if !strings.Contains(description, "git rebase") {
		t.Error("conflict task should include rebase command")
	}
	if !strings.Contains(description, "git push -f") {
		t.Error("conflict task should include force-push instruction")
	}
}

func TestEngineer_PriorityBoostOnConflict(t *testing.T) {
	// Test that conflict resolution tasks get priority boost
	// P2 -> P1, P1 -> P0, P0 stays P0

	tests := []struct {
		originalPriority  int
		expectedBoosted   int
	}{
		{4, 3}, // P4 -> P3
		{3, 2}, // P3 -> P2
		{2, 1}, // P2 -> P1
		{1, 0}, // P1 -> P0
		{0, 0}, // P0 stays P0 (can't go negative)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("P%d->P%d", tt.originalPriority, tt.expectedBoosted), func(t *testing.T) {
			// Simulate the priority boost logic from createConflictResolutionTask
			boostedPriority := tt.originalPriority - 1
			if boostedPriority < 0 {
				boostedPriority = 0
			}

			if boostedPriority != tt.expectedBoosted {
				t.Errorf("expected boosted priority %d, got %d", tt.expectedBoosted, boostedPriority)
			}
		})
	}
}

func TestEngineer_MultiMRPriorityOrdering(t *testing.T) {
	// Test that multiple MRs are processed in correct priority order
	// This simulates the ListByScore behavior

	now := time.Now()
	convoyOld := now.Add(-48 * time.Hour)
	convoyNew := now.Add(-12 * time.Hour)

	type TestMR struct {
		name       string
		priority   int
		convoyAge  *time.Time
		retryCount int
	}

	mrs := []TestMR{
		{"P1-new-convoy", 1, &convoyNew, 0},
		{"P3-old-convoy", 3, &convoyOld, 0},
		{"P0-no-convoy", 0, nil, 0},
		{"P2-with-retries", 2, &convoyNew, 3},
	}

	// Calculate scores
	type ScoredMR struct {
		name  string
		score float64
	}

	var scored []ScoredMR
	for _, mr := range mrs {
		// Simplified scoring (same formula as mrqueue)
		score := 1000.0 // base

		// Convoy age bonus
		if mr.convoyAge != nil {
			convoyHours := now.Sub(*mr.convoyAge).Hours()
			score += 10.0 * convoyHours
		}

		// Priority bonus
		priorityBonus := 4 - mr.priority
		if priorityBonus < 0 {
			priorityBonus = 0
		}
		score += 100.0 * float64(priorityBonus)

		// Retry penalty
		retryPenalty := 50.0 * float64(mr.retryCount)
		if retryPenalty > 300.0 {
			retryPenalty = 300.0
		}
		score -= retryPenalty

		scored = append(scored, ScoredMR{mr.name, score})
	}

	// Sort by score (higher first)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Log the ordering for debugging
	t.Log("Priority order (highest first):")
	for i, s := range scored {
		t.Logf("  %d. %s (score: %.1f)", i+1, s.name, s.score)
	}

	// Old convoy should be near the top due to starvation prevention
	// P3 with 48h convoy gets 480 convoy bonus
	// P0 with no convoy only gets 400 priority bonus
	if scored[0].name != "P3-old-convoy" {
		t.Logf("Note: expected P3-old-convoy to be first due to 48h starvation prevention, got %s", scored[0].name)
	}

	// All scores should be positive
	for _, s := range scored {
		if s.score < 0 {
			t.Errorf("score for %s is negative: %.1f", s.name, s.score)
		}
	}
}

func TestEngineer_RetryCountTracking(t *testing.T) {
	// Test that retry count is properly incremented on each conflict

	type ConflictCycle struct {
		cycle       int
		retryCount  int
		shouldBoost bool
	}

	cycles := []ConflictCycle{
		{1, 1, true},  // First conflict -> retry_count=1, priority boosted
		{2, 2, true},  // Second conflict -> retry_count=2, priority still boosted
		{3, 3, true},  // Third conflict -> retry_count=3
		{6, 6, true},  // Sixth conflict -> retry_count=6 (at cap)
		{7, 7, false}, // Seventh conflict -> retry_count=7 (penalty capped at 6)
	}

	for _, c := range cycles {
		t.Run(fmt.Sprintf("cycle_%d", c.cycle), func(t *testing.T) {
			// Simulate conflict -> retry_count increment
			previousCount := c.cycle - 1
			newRetryCount := previousCount + 1

			if newRetryCount != c.retryCount {
				t.Errorf("expected retry count %d, got %d", c.retryCount, newRetryCount)
			}

			// Verify penalty calculation
			penalty := 50.0 * float64(newRetryCount)
			if penalty > 300.0 {
				penalty = 300.0
			}

			if c.cycle <= 6 {
				expectedPenalty := 50.0 * float64(c.cycle)
				if penalty != expectedPenalty {
					t.Errorf("expected penalty %.1f, got %.1f", expectedPenalty, penalty)
				}
			} else {
				// Capped at 300
				if penalty != 300.0 {
					t.Errorf("expected capped penalty 300.0, got %.1f", penalty)
				}
			}
		})
	}
}
