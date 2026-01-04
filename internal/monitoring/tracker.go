package monitoring

import (
	"sync"
	"time"

	"github.com/steveyegge/gastown/internal/tmux"
)

// Tracker monitors and infers status for multiple agents.
type Tracker struct {
	mu          sync.RWMutex
	detector    *PatternDetector
	tmux        *tmux.Tmux
	statuses    map[string]*AgentStatus // Address -> AgentStatus
	lastCapture map[string]time.Time    // Track when pane was last captured
}

// NewTracker creates a new agent status tracker.
func NewTracker(t *tmux.Tmux) *Tracker {
	if t == nil {
		t = tmux.NewTmux()
	}
	return &Tracker{
		detector:    NewPatternDetector(),
		tmux:        t,
		statuses:    make(map[string]*AgentStatus),
		lastCapture: make(map[string]time.Time),
	}
}

// InferStatus infers the current status of an agent from its tmux pane output.
// Checks for common patterns like "Thinking...", "BLOCKED:", "Error:", etc.
func (tr *Tracker) InferStatus(address string, tmuxSession string, agentState string, hasWork bool) *AgentStatus {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	status := &AgentStatus{
		Address:      address,
		LastUpdated:  time.Now(),
		AgentState:   agentState,
		HasWork:      hasWork,
		LastActivity: time.Now(),
	}

	// Check if tmux session exists
	exists, err := tr.tmux.HasSession(tmuxSession)
	if err != nil || !exists {
		status.Status = StatusOffline
		status.Source = SourceInferred
		tr.statuses[address] = status
		tr.detector.ResetActivity(address)
		return status
	}

	status.Running = true

	// Capture pane output (last 30 lines)
	paneOutput, err := tr.tmux.CapturePane(tmuxSession, 30)
	if err != nil {
		// Session exists but can't capture - likely offline or error
		status.Status = StatusError
		status.Source = SourceInferred
		tr.statuses[address] = status
		return status
	}

	status.PaneOutput = paneOutput

	// Detect status from pane output patterns
	detectedStatus, pattern := tr.detector.DetectStatus(address, paneOutput)
	if detectedStatus != "" {
		status.Status = detectedStatus
		status.MatchedPattern = pattern
		status.Source = SourceInferred
		tr.statuses[address] = status
		return status
	}

	// Check for idle status
	now := time.Now()
	if idleStatus, idleDuration := tr.detector.GetIdleStatus(address, now); idleStatus == StatusIdle {
		status.Status = StatusIdle
		status.IdleDuration = idleDuration
		status.Source = SourceInferred
		tr.statuses[address] = status
		return status
	}

	// No specific pattern matched and not idle - infer from agent state
	status.Status = inferStatusFromAgentState(agentState, hasWork)
	status.Source = SourceInferred

	tr.statuses[address] = status
	return status
}

// SetStatus explicitly sets an agent's status (from boss like Witness).
func (tr *Tracker) SetStatus(address string, status string, source SourceType) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	agentStatus := &AgentStatus{
		Address:     address,
		Status:      status,
		Source:      source,
		LastUpdated: time.Now(),
	}

	tr.statuses[address] = agentStatus
}

// GetStatus retrieves the current status for an agent.
func (tr *Tracker) GetStatus(address string) *AgentStatus {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	if status, ok := tr.statuses[address]; ok {
		return status
	}
	return nil
}

// GetAllStatuses returns all tracked agent statuses.
func (tr *Tracker) GetAllStatuses() []*AgentStatus {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	statuses := make([]*AgentStatus, 0, len(tr.statuses))
	for _, status := range tr.statuses {
		statuses = append(statuses, status)
	}
	return statuses
}

// Clear removes all tracked statuses.
func (tr *Tracker) Clear() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.statuses = make(map[string]*AgentStatus)
	tr.lastCapture = make(map[string]time.Time)
	tr.detector = NewPatternDetector()
}

// inferStatusFromAgentState infers a status based on agent state from beads.
// This is used when no pattern is detected in pane output.
func inferStatusFromAgentState(agentState string, hasWork bool) string {
	switch agentState {
	case "spawning":
		return StatusWorking
	case "working":
		return StatusWorking
	case "done":
		return StatusAvailable
	case "stuck":
		return StatusBlocked
	default:
		// Fallback to work status
		if hasWork {
			return StatusWorking
		}
		return StatusAvailable
	}
}
