package monitoring

import (
	"testing"
	"time"
)

func TestPatternDetector_DetectStatus(t *testing.T) {
	pd := NewPatternDetector()

	tests := []struct {
		name          string
		input         string
		expectedStatus string
		expectedPattern string
	}{
		{
			name:          "detect thinking",
			input:         "Thinking...",
			expectedStatus: StatusThinking,
			expectedPattern: "Thinking",
		},
		{
			name:          "detect blocked",
			input:         "BLOCKED: waiting for upstream",
			expectedStatus: StatusBlocked,
			expectedPattern: "BLOCKED",
		},
		{
			name:          "detect error",
			input:         "Error: failed to compile",
			expectedStatus: StatusError,
			expectedPattern: "Error",
		},
		{
			name:          "detect waiting",
			input:         "Waiting for response from server",
			expectedStatus: StatusBlocked,
			expectedPattern: "Waiting",
		},
		{
			name:          "detect running",
			input:         "Running tests...",
			expectedStatus: StatusWorking,
			expectedPattern: "Running",
		},
		{
			name:          "detect panic",
			input:         "panic: null pointer exception",
			expectedStatus: StatusError,
			expectedPattern: "Panic",
		},
		{
			name:          "empty output",
			input:         "",
			expectedStatus: "",
			expectedPattern: "",
		},
		{
			name:          "whitespace only",
			input:         "   \n  \t  ",
			expectedStatus: "",
			expectedPattern: "",
		},
		{
			name:          "case insensitive",
			input:         "ERROR: something went wrong",
			expectedStatus: StatusError,
			expectedPattern: "Error",
		},
		{
			name:          "no pattern match",
			input:         "random output text",
			expectedStatus: StatusWorking,
			expectedPattern: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, pattern := pd.DetectStatus("test-agent", tt.input)
			if status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, status)
			}
			if pattern != tt.expectedPattern {
				t.Errorf("expected pattern %q, got %q", tt.expectedPattern, pattern)
			}
		})
	}
}

func TestPatternDetector_ActivityTracking(t *testing.T) {
	pd := NewPatternDetector()
	address := "test-agent"

	// Initially, no activity recorded
	_, exists := pd.LastActivity(address)
	if exists {
		t.Error("expected no activity recorded initially")
	}

	// Detect status should record activity
	pd.DetectStatus(address, "Thinking...")
	_, exists2 := pd.LastActivity(address)
	if !exists2 {
		t.Error("expected activity to be recorded")
	}

	// Idle status should be based on last activity
	now := time.Now().Add(IdleThreshold + 1*time.Second)
	idleStatus, _ := pd.GetIdleStatus(address, now)
	if idleStatus != StatusIdle {
		t.Errorf("expected idle status, got %q", idleStatus)
	}

	// Reset activity
	pd.ResetActivity(address)
	_, exists = pd.LastActivity(address)
	if exists {
		t.Error("expected activity to be cleared after reset")
	}
}

func TestPatternDetector_IdleDetection(t *testing.T) {
	pd := NewPatternDetector()
	address := "test-agent"

	// Record initial activity
	pd.RecordActivity(address)
	now := time.Now()

	// Immediately after activity - not idle
	idleStatus, _ := pd.GetIdleStatus(address, now)
	if idleStatus == StatusIdle {
		t.Errorf("expected non-idle status immediately after activity, got %q", idleStatus)
	}

	// After threshold - should be idle
	laterTime := now.Add(IdleThreshold + 1*time.Second)
	idleStatus, idleDuration := pd.GetIdleStatus(address, laterTime)
	if idleStatus != StatusIdle {
		t.Errorf("expected idle status after threshold, got %q", idleStatus)
	}
	if idleDuration < IdleThreshold {
		t.Errorf("expected idle duration >= threshold, got %v", idleDuration)
	}
}

func TestPatternDetector_MultipleAgents(t *testing.T) {
	pd := NewPatternDetector()

	// Track multiple agents
	agent1 := "agent-1"
	agent2 := "agent-2"

	pd.DetectStatus(agent1, "Thinking...")
	pd.DetectStatus(agent2, "Error: failed")

	// Both should have activity recorded
	_, exists1 := pd.LastActivity(agent1)
	_, exists2 := pd.LastActivity(agent2)

	if !exists1 || !exists2 {
		t.Error("expected both agents to have activity recorded")
	}

	// Check idle status at same time point - both should be active (just recorded)
	now := time.Now()
	idleStatus1, _ := pd.GetIdleStatus(agent1, now)
	idleStatus2, _ := pd.GetIdleStatus(agent2, now)

	if idleStatus1 == StatusIdle || idleStatus2 == StatusIdle {
		t.Error("expected both agents to be non-idle immediately after activity")
	}
}
