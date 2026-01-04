package monitoring

import (
	"testing"
	"time"
)

func TestTracker_InferStatus_Offline(t *testing.T) {
	// Create a mock tmux that doesn't find the session
	tr := NewTracker(nil)

	// Since we're using a real tmux instance, this will try to check a non-existent session
	// and return offline status
	status := tr.InferStatus("test-agent", "non-existent-session", "", false)

	if status.Status != StatusOffline {
		t.Errorf("expected offline status, got %q", status.Status)
	}
	if status.Source != SourceInferred {
		t.Errorf("expected inferred source, got %q", status.Source)
	}
	if status.Running {
		t.Error("expected running to be false for offline agent")
	}
}

func TestTracker_SetAndGetStatus(t *testing.T) {
	tr := NewTracker(nil)
	address := "test-rig/test-agent"

	// Set explicit status
	tr.SetStatus(address, StatusWorking, SourceBoss)

	// Retrieve status
	status := tr.GetStatus(address)
	if status == nil {
		t.Fatal("expected status to be set")
	}
	if status.Status != StatusWorking {
		t.Errorf("expected working status, got %q", status.Status)
	}
	if status.Source != SourceBoss {
		t.Errorf("expected boss source, got %q", status.Source)
	}
}

func TestTracker_InferStatusFromAgentState(t *testing.T) {
	tests := []struct {
		name           string
		agentState     string
		hasWork        bool
		expectedStatus string
	}{
		{
			name:           "spawning state",
			agentState:     "spawning",
			hasWork:        false,
			expectedStatus: StatusWorking,
		},
		{
			name:           "working state with work",
			agentState:     "working",
			hasWork:        true,
			expectedStatus: StatusWorking,
		},
		{
			name:           "done state",
			agentState:     "done",
			hasWork:        false,
			expectedStatus: StatusAvailable,
		},
		{
			name:           "stuck state",
			agentState:     "stuck",
			hasWork:        true,
			expectedStatus: StatusBlocked,
		},
		{
			name:           "unknown state with work",
			agentState:     "unknown",
			hasWork:        true,
			expectedStatus: StatusWorking,
		},
		{
			name:           "unknown state no work",
			agentState:     "unknown",
			hasWork:        false,
			expectedStatus: StatusAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the internal state inference directly
			inferred := inferStatusFromAgentState(tt.agentState, tt.hasWork)
			if inferred != tt.expectedStatus {
				t.Errorf("expected %q, got %q", tt.expectedStatus, inferred)
			}
		})
	}
}

func TestTracker_GetAllStatuses(t *testing.T) {
	tr := NewTracker(nil)

	// Set multiple statuses
	tr.SetStatus("agent-1", StatusWorking, SourceBoss)
	tr.SetStatus("agent-2", StatusIdle, SourceBoss)
	tr.SetStatus("agent-3", StatusBlocked, SourceBoss)

	// Get all statuses
	allStatuses := tr.GetAllStatuses()

	if len(allStatuses) != 3 {
		t.Errorf("expected 3 statuses, got %d", len(allStatuses))
	}

	// Verify we can find each status
	statusMap := make(map[string]*AgentStatus)
	for _, s := range allStatuses {
		statusMap[s.Address] = s
	}

	if _, ok := statusMap["agent-1"]; !ok {
		t.Error("expected agent-1 in results")
	}
	if _, ok := statusMap["agent-2"]; !ok {
		t.Error("expected agent-2 in results")
	}
	if _, ok := statusMap["agent-3"]; !ok {
		t.Error("expected agent-3 in results")
	}
}

func TestTracker_Clear(t *testing.T) {
	tr := NewTracker(nil)

	// Set some statuses
	tr.SetStatus("agent-1", StatusWorking, SourceBoss)
	tr.SetStatus("agent-2", StatusIdle, SourceBoss)

	// Verify they're set
	allStatuses := tr.GetAllStatuses()
	if len(allStatuses) != 2 {
		t.Errorf("expected 2 statuses before clear, got %d", len(allStatuses))
	}

	// Clear
	tr.Clear()

	// Verify they're gone
	allStatuses = tr.GetAllStatuses()
	if len(allStatuses) != 0 {
		t.Errorf("expected 0 statuses after clear, got %d", len(allStatuses))
	}
}

func TestTracker_StatusTimestamp(t *testing.T) {
	tr := NewTracker(nil)
	address := "test-agent"

	before := time.Now()
	tr.SetStatus(address, StatusWorking, SourceBoss)
	after := time.Now()

	status := tr.GetStatus(address)
	if status == nil {
		t.Fatal("expected status to be set")
	}

	if status.LastUpdated.Before(before) || status.LastUpdated.After(after.Add(1*time.Second)) {
		t.Errorf("expected LastUpdated to be between %v and %v, got %v", before, after, status.LastUpdated)
	}
}
