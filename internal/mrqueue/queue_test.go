package mrqueue

import (
	"os"
	"testing"
	"time"
)

func TestQueue_SubmitAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	mr := &MR{
		Branch:      "polecat/test",
		Target:      "main",
		SourceIssue: "gt-abc",
		Worker:      "test-worker",
		Rig:         "test-rig",
		Priority:    2,
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	if mr.ID == "" {
		t.Error("Expected ID to be set after Submit")
	}

	got, err := q.Get(mr.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Branch != mr.Branch {
		t.Errorf("Branch mismatch: got %s, want %s", got.Branch, mr.Branch)
	}
	if got.Target != mr.Target {
		t.Errorf("Target mismatch: got %s, want %s", got.Target, mr.Target)
	}
}

func TestQueue_Update(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	// Submit initial MR
	mr := &MR{
		Branch:      "polecat/test",
		Target:      "main",
		SourceIssue: "gt-abc",
		Worker:      "test-worker",
		Rig:         "test-rig",
		Priority:    2,
		RetryCount:  0,
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Update with conflict metadata
	mr.RetryCount = 1
	mr.LastConflictSHA = "abc123def456"
	mr.ConflictTaskID = "gt-conflict-task"

	if err := q.Update(mr); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Read back and verify
	got, err := q.Get(mr.ID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}

	if got.RetryCount != 1 {
		t.Errorf("RetryCount: got %d, want 1", got.RetryCount)
	}
	if got.LastConflictSHA != "abc123def456" {
		t.Errorf("LastConflictSHA: got %s, want abc123def456", got.LastConflictSHA)
	}
	if got.ConflictTaskID != "gt-conflict-task" {
		t.Errorf("ConflictTaskID: got %s, want gt-conflict-task", got.ConflictTaskID)
	}
}

func TestQueue_Update_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)
	if err := q.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Try to update non-existent MR
	mr := &MR{
		ID:     "mr-does-not-exist",
		Branch: "test",
	}

	err = q.Update(mr)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestQueue_ConflictMetadataInList(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	// Submit MR with conflict metadata
	mr := &MR{
		Branch:          "polecat/test",
		Target:          "main",
		SourceIssue:     "gt-abc",
		Worker:          "test-worker",
		Rig:             "test-rig",
		Priority:        2,
		RetryCount:      2,
		LastConflictSHA: "sha123",
		ConflictTaskID:  "gt-task-123",
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// List should include conflict metadata
	mrs, err := q.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(mrs) != 1 {
		t.Fatalf("Expected 1 MR, got %d", len(mrs))
	}

	if mrs[0].RetryCount != 2 {
		t.Errorf("RetryCount in list: got %d, want 2", mrs[0].RetryCount)
	}
	if mrs[0].LastConflictSHA != "sha123" {
		t.Errorf("LastConflictSHA in list: got %s, want sha123", mrs[0].LastConflictSHA)
	}
	if mrs[0].ConflictTaskID != "gt-task-123" {
		t.Errorf("ConflictTaskID in list: got %s, want gt-task-123", mrs[0].ConflictTaskID)
	}
}

func TestQueue_Remove(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	mr := &MR{
		Branch: "polecat/test",
		Target: "main",
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Verify it exists
	if q.Count() != 1 {
		t.Errorf("Expected count 1, got %d", q.Count())
	}

	// Remove it
	if err := q.Remove(mr.ID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	if q.Count() != 0 {
		t.Errorf("Expected count 0 after remove, got %d", q.Count())
	}

	// Remove again should not error
	if err := q.Remove(mr.ID); err != nil {
		t.Errorf("Second remove should not error, got %v", err)
	}
}

func TestQueue_Claim(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	mr := &MR{
		Branch: "polecat/test",
		Target: "main",
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Claim it
	if err := q.Claim(mr.ID, "worker-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Verify claim
	got, err := q.Get(mr.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ClaimedBy != "worker-1" {
		t.Errorf("ClaimedBy: got %s, want worker-1", got.ClaimedBy)
	}
	if got.ClaimedAt == nil {
		t.Error("ClaimedAt should be set")
	}

	// Another worker should fail
	err = q.Claim(mr.ID, "worker-2")
	if err != ErrAlreadyClaimed {
		t.Errorf("Expected ErrAlreadyClaimed, got %v", err)
	}
}

func TestQueue_Release(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	mr := &MR{
		Branch: "polecat/test",
		Target: "main",
	}

	if err := q.Submit(mr); err != nil {
		t.Fatalf("Submit failed: %v", err)
	}

	// Claim it
	if err := q.Claim(mr.ID, "worker-1"); err != nil {
		t.Fatalf("Claim failed: %v", err)
	}

	// Release it
	if err := q.Release(mr.ID); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Verify release
	got, err := q.Get(mr.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ClaimedBy != "" {
		t.Errorf("ClaimedBy should be empty after release, got %s", got.ClaimedBy)
	}
	if got.ClaimedAt != nil {
		t.Error("ClaimedAt should be nil after release")
	}

	// Another worker can now claim
	if err := q.Claim(mr.ID, "worker-2"); err != nil {
		t.Errorf("Claim after release should succeed, got %v", err)
	}
}

func TestQueue_ListByScore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mrqueue-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	q := New(tmpDir)

	// Submit MRs with different priorities
	now := time.Now()
	mrs := []*MR{
		{Branch: "low", Target: "main", Priority: 4, CreatedAt: now},   // Low priority
		{Branch: "high", Target: "main", Priority: 0, CreatedAt: now},  // High priority
		{Branch: "med", Target: "main", Priority: 2, CreatedAt: now},   // Medium priority
	}

	for _, mr := range mrs {
		if err := q.Submit(mr); err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}

	// List by score should return high priority first
	sorted, err := q.ListByScore()
	if err != nil {
		t.Fatalf("ListByScore failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 MRs, got %d", len(sorted))
	}

	// Verify order: P0 (high) > P2 (med) > P4 (low)
	if sorted[0].Branch != "high" {
		t.Errorf("First should be 'high', got %s", sorted[0].Branch)
	}
	if sorted[1].Branch != "med" {
		t.Errorf("Second should be 'med', got %s", sorted[1].Branch)
	}
	if sorted[2].Branch != "low" {
		t.Errorf("Third should be 'low', got %s", sorted[2].Branch)
	}
}
