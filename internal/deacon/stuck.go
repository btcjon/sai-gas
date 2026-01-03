// Package deacon provides the Deacon agent infrastructure.
package deacon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default parameters for stuck-session detection.
const (
	DefaultPingTimeout        = 30 * time.Second // How long to wait for response
	DefaultConsecutiveFailures = 3               // Failures before force-kill
	DefaultCooldown           = 5 * time.Minute  // Minimum time between force-kills
)

// StuckConfig holds configurable parameters for stuck-session detection.
type StuckConfig struct {
	PingTimeout         time.Duration `json:"ping_timeout"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	Cooldown            time.Duration `json:"cooldown"`
}

// DefaultStuckConfig returns the default stuck detection config.
func DefaultStuckConfig() *StuckConfig {
	return &StuckConfig{
		PingTimeout:         DefaultPingTimeout,
		ConsecutiveFailures: DefaultConsecutiveFailures,
		Cooldown:            DefaultCooldown,
	}
}

// AgentHealthState tracks the health check state for a single agent.
type AgentHealthState struct {
	// AgentID is the identifier (e.g., "gastown/polecats/max" or "deacon")
	AgentID string `json:"agent_id"`

	// LastPingTime is when we last sent a HEALTH_CHECK nudge
	LastPingTime time.Time `json:"last_ping_time,omitempty"`

	// LastResponseTime is when the agent last updated their activity
	LastResponseTime time.Time `json:"last_response_time,omitempty"`

	// ConsecutiveFailures counts how many health checks failed in a row
	ConsecutiveFailures int `json:"consecutive_failures"`

	// LastForceKillTime is when we last force-killed this agent
	LastForceKillTime time.Time `json:"last_force_kill_time,omitempty"`

	// ForceKillCount is total number of force-kills for this agent
	ForceKillCount int `json:"force_kill_count"`
}

// HealthCheckState holds health check state for all monitored agents.
type HealthCheckState struct {
	// Agents maps agent ID to their health state
	Agents map[string]*AgentHealthState `json:"agents"`

	// LastUpdated is when this state was last written
	LastUpdated time.Time `json:"last_updated"`
}

// HealthCheckStateFile returns the path to the health check state file.
func HealthCheckStateFile(townRoot string) string {
	return filepath.Join(townRoot, "deacon", "health-check-state.json")
}

// LoadHealthCheckState loads the health check state from disk.
// Returns empty state if file doesn't exist.
func LoadHealthCheckState(townRoot string) (*HealthCheckState, error) {
	stateFile := HealthCheckStateFile(townRoot)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state
			return &HealthCheckState{
				Agents: make(map[string]*AgentHealthState),
			}, nil
		}
		return nil, fmt.Errorf("reading health check state: %w", err)
	}

	var state HealthCheckState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing health check state: %w", err)
	}

	if state.Agents == nil {
		state.Agents = make(map[string]*AgentHealthState)
	}

	return &state, nil
}

// SaveHealthCheckState saves the health check state to disk.
func SaveHealthCheckState(townRoot string, state *HealthCheckState) error {
	stateFile := HealthCheckStateFile(townRoot)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
		return fmt.Errorf("creating deacon directory: %w", err)
	}

	state.LastUpdated = time.Now().UTC()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling health check state: %w", err)
	}

	return os.WriteFile(stateFile, data, 0644)
}

// GetAgentState returns the health state for an agent, creating if needed.
func (s *HealthCheckState) GetAgentState(agentID string) *AgentHealthState {
	if s.Agents == nil {
		s.Agents = make(map[string]*AgentHealthState)
	}

	state, ok := s.Agents[agentID]
	if !ok {
		state = &AgentHealthState{AgentID: agentID}
		s.Agents[agentID] = state
	}
	return state
}

// HealthCheckResult represents the outcome of a health check.
type HealthCheckResult struct {
	AgentID             string        `json:"agent_id"`
	Responded           bool          `json:"responded"`
	ResponseTime        time.Duration `json:"response_time,omitempty"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	ShouldForceKill     bool          `json:"should_force_kill"`
	InCooldown          bool          `json:"in_cooldown"`
	CooldownRemaining   time.Duration `json:"cooldown_remaining,omitempty"`
}

// Common errors for stuck-session detection.
var (
	ErrAgentInCooldown  = errors.New("agent is in cooldown period after recent force-kill")
	ErrAgentNotFound    = errors.New("agent not found or session doesn't exist")
	ErrAgentResponsive  = errors.New("agent is responsive, no action needed")
)

// RecordPing records that a health check ping was sent to an agent.
func (s *AgentHealthState) RecordPing() {
	s.LastPingTime = time.Now().UTC()
}

// RecordResponse records that an agent responded to a health check.
// This resets the consecutive failure counter.
func (s *AgentHealthState) RecordResponse() {
	s.LastResponseTime = time.Now().UTC()
	s.ConsecutiveFailures = 0
}

// RecordFailure records that an agent failed to respond to a health check.
func (s *AgentHealthState) RecordFailure() {
	s.ConsecutiveFailures++
}

// RecordForceKill records that an agent was force-killed.
func (s *AgentHealthState) RecordForceKill() {
	s.LastForceKillTime = time.Now().UTC()
	s.ForceKillCount++
	s.ConsecutiveFailures = 0 // Reset after kill
}

// IsInCooldown returns true if the agent was recently force-killed.
func (s *AgentHealthState) IsInCooldown(cooldown time.Duration) bool {
	if s.LastForceKillTime.IsZero() {
		return false
	}
	return time.Since(s.LastForceKillTime) < cooldown
}

// CooldownRemaining returns how long until cooldown expires.
func (s *AgentHealthState) CooldownRemaining(cooldown time.Duration) time.Duration {
	if s.LastForceKillTime.IsZero() {
		return 0
	}
	remaining := cooldown - time.Since(s.LastForceKillTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ShouldForceKill returns true if the agent has exceeded the failure threshold.
func (s *AgentHealthState) ShouldForceKill(threshold int) bool {
	return s.ConsecutiveFailures >= threshold
}

// WatchdogDecision represents the outcome of evaluating an agent's health.
type WatchdogDecision struct {
	AgentID         string        `json:"agent_id"`
	Action          string        `json:"action"` // "allow", "block", "force_kill"
	Reason          string        `json:"reason"`
	CircuitState    CircuitState  `json:"circuit_state"`
	RetryAfter      time.Duration `json:"retry_after,omitempty"`
	ConsecFailures  int           `json:"consecutive_failures"`
	SuccessRate     float64       `json:"success_rate"`
}

// EvaluateAgent combines health check state and circuit breaker state
// to make a decision about how to handle an agent.
// This is the main integration point for the watchdog loop.
func EvaluateAgent(
	agentID string,
	healthState *HealthCheckState,
	cbRegistry *CircuitBreakerRegistry,
	stuckCfg *StuckConfig,
) *WatchdogDecision {
	agent := healthState.GetAgentState(agentID)
	cb := cbRegistry.GetBreaker(agentID)
	cbCfg := cbRegistry.GetConfig()

	decision := &WatchdogDecision{
		AgentID:        agentID,
		CircuitState:   cb.State,
		ConsecFailures: agent.ConsecutiveFailures,
		SuccessRate:    cb.SuccessRate(),
	}

	// Check circuit breaker first - if open, we block
	if !cb.ShouldAllow(cbCfg) {
		decision.Action = "block"
		decision.Reason = "circuit breaker open"
		decision.RetryAfter = cb.TimeUntilRetry(cbCfg)
		return decision
	}

	// Check if in cooldown
	if agent.IsInCooldown(stuckCfg.Cooldown) {
		decision.Action = "block"
		decision.Reason = "cooldown after force-kill"
		decision.RetryAfter = agent.CooldownRemaining(stuckCfg.Cooldown)
		return decision
	}

	// Check if should force-kill
	if agent.ShouldForceKill(stuckCfg.ConsecutiveFailures) {
		decision.Action = "force_kill"
		decision.Reason = "exceeded failure threshold"
		return decision
	}

	// Allow the operation
	decision.Action = "allow"
	decision.Reason = "healthy"
	return decision
}

// RecordHealthCheckOutcome updates both health state and circuit breaker
// based on the outcome of a health check.
func RecordHealthCheckOutcome(
	agentID string,
	success bool,
	healthState *HealthCheckState,
	cbRegistry *CircuitBreakerRegistry,
) (stateChanged bool) {
	agent := healthState.GetAgentState(agentID)
	cb := cbRegistry.GetBreaker(agentID)
	cbCfg := cbRegistry.GetConfig()

	if success {
		agent.RecordResponse()
		stateChanged = cb.RecordSuccess(cbCfg)
	} else {
		agent.RecordFailure()
		stateChanged = cb.RecordFailure(cbCfg)
	}

	return stateChanged
}

// RecordForceKillOutcome records that a force-kill was performed.
// Updates both health state and circuit breaker.
func RecordForceKillOutcome(
	agentID string,
	healthState *HealthCheckState,
	cbRegistry *CircuitBreakerRegistry,
) {
	agent := healthState.GetAgentState(agentID)
	agent.RecordForceKill()

	// A force-kill counts as a failure from the circuit breaker's perspective
	cb := cbRegistry.GetBreaker(agentID)
	cbCfg := cbRegistry.GetConfig()
	cb.RecordFailure(cbCfg)
}

// WatchdogSummary provides a summary of the overall system health.
type WatchdogSummary struct {
	TotalAgents      int                  `json:"total_agents"`
	HealthyAgents    int                  `json:"healthy_agents"`
	UnhealthyAgents  int                  `json:"unhealthy_agents"`
	CircuitStates    map[CircuitState]int `json:"circuit_states"`
	TotalForceKills  int                  `json:"total_force_kills"`
	AverageSuccessRate float64            `json:"average_success_rate"`
}

// GetWatchdogSummary generates a summary of current system health.
func GetWatchdogSummary(
	healthState *HealthCheckState,
	cbRegistry *CircuitBreakerRegistry,
	stuckCfg *StuckConfig,
) *WatchdogSummary {
	summary := &WatchdogSummary{
		CircuitStates: cbRegistry.Summary(),
	}

	var totalSuccessRate float64
	for agentID := range healthState.Agents {
		summary.TotalAgents++
		agent := healthState.Agents[agentID]
		summary.TotalForceKills += agent.ForceKillCount

		cb := cbRegistry.GetBreaker(agentID)
		totalSuccessRate += cb.SuccessRate()

		// Determine if healthy
		decision := EvaluateAgent(agentID, healthState, cbRegistry, stuckCfg)
		if decision.Action == "allow" {
			summary.HealthyAgents++
		} else {
			summary.UnhealthyAgents++
		}
	}

	if summary.TotalAgents > 0 {
		summary.AverageSuccessRate = totalSuccessRate / float64(summary.TotalAgents)
	} else {
		summary.AverageSuccessRate = 100.0
	}

	return summary
}
