// Package monitoring provides agent status monitoring and inference.
package monitoring

import "time"

// Status types for agent state
const (
	StatusAvailable = "available"
	StatusWorking   = "working"
	StatusThinking  = "thinking"
	StatusBlocked   = "blocked"
	StatusIdle      = "idle"
	StatusError     = "error"
	StatusOffline   = "offline"
)

// SourceType indicates where the status came from
type SourceType string

const (
	SourceBoss     SourceType = "boss"      // Witness/Mayor set explicitly
	SourceSelf     SourceType = "self"      // Agent reports own status
	SourceInferred SourceType = "inferred"  // Detected from pane output patterns
)

// AgentStatus represents the current status of an agent.
type AgentStatus struct {
	// Agent identification
	Address string // Full address (e.g., "rig-name/polecat-name")
	Role    string // polecat, crew, witness, refinery

	// Status information
	Status        string     // Current status (available, working, etc.)
	Source        SourceType // Where status came from
	LastUpdated   time.Time  // When status was last updated
	LastActivity  time.Time  // Last detected activity
	IdleDuration  time.Duration // How long idle (if StatusIdle)

	// Activity details
	PaneOutput    string // Last captured pane output
	MatchedPattern string // Which pattern (if any) matched in pane output

	// Agent state from beads
	AgentState   string // spawning, working, done, stuck
	HookBead     string // Currently pinned work bead ID
	HasWork      bool   // Has pinned work
	WorkTitle    string // Title of pinned work
	Running      bool   // Is tmux session running
}

// StatusReport is the result of status inference for a single agent.
type StatusReport struct {
	Status *AgentStatus
	Error  error
}

// IdleThreshold is the duration of no output before marking an agent as idle
const IdleThreshold = 60 * time.Second
