// Package polecat provides polecat lifecycle management.
package polecat

import "time"

// State represents the current state of a polecat.
// In the transient model, polecats exist only while working.
type State string

const (
	// StateWorking means the polecat is actively working on an issue.
	// This is the initial and primary state for transient polecats.
	StateWorking State = "working"

	// StateMRSubmitted means the polecat has submitted its MR to the queue.
	// In the ephemeral model, this is a transitional state between working and recyclable.
	// The polecat is waiting for the refinery to acknowledge the MR is queued.
	// State transitions:
	//   working → mr_submitted (after `gt done --exit COMPLETED` creates MR)
	//   mr_submitted → recyclable (after branch confirmed pushed to origin)
	StateMRSubmitted State = "mr_submitted"

	// StateRecyclable means the polecat has submitted its MR and is ready
	// for cleanup. This is the ephemeral exit state set by `gt done --exit COMPLETED`.
	// The polecat has pushed all work to origin and can be safely nuked.
	// The Witness will nuke the polecat once the MERGED signal is received.
	StateRecyclable State = "recyclable"

	// StateDone means the polecat has completed its assigned work
	// and is ready for cleanup by the Witness.
	// Deprecated: use StateRecyclable for ephemeral polecats
	StateDone State = "done"

	// StateStuck means the polecat needs assistance.
	StateStuck State = "stuck"

	// Legacy states for backward compatibility during transition.
	// New code should not use these.
	StateIdle   State = "idle"   // Deprecated: use StateWorking
	StateActive State = "active" // Deprecated: use StateWorking
)

// IsWorking returns true if the polecat is currently working.
func (s State) IsWorking() bool {
	return s == StateWorking
}

// IsActive returns true if the polecat session is actively working.
// For transient polecats, this is true for working state and
// legacy idle/active states (treated as working).
func (s State) IsActive() bool {
	return s == StateWorking || s == StateIdle || s == StateActive
}

// IsRecyclable returns true if the polecat has submitted its MR
// and is ready to be nuked. This is the clean exit state for
// ephemeral polecats.
func (s State) IsRecyclable() bool {
	return s == StateRecyclable
}

// IsMRSubmitted returns true if the polecat has submitted its MR
// but hasn't been confirmed recyclable yet. This is the transitional
// state in the ephemeral model.
func (s State) IsMRSubmitted() bool {
	return s == StateMRSubmitted
}

// IsReadyForCleanup returns true if the polecat is in a state where
// the Witness can consider cleaning it up. This includes:
//   - mr_submitted: MR submitted, waiting for confirmation
//   - recyclable: Ready for immediate cleanup
//   - done: Legacy state, equivalent to recyclable
func (s State) IsReadyForCleanup() bool {
	return s == StateMRSubmitted || s == StateRecyclable || s == StateDone
}

// Polecat represents a worker agent in a rig.
type Polecat struct {
	// Name is the polecat identifier.
	Name string `json:"name"`

	// Rig is the rig this polecat belongs to.
	Rig string `json:"rig"`

	// State is the current lifecycle state.
	State State `json:"state"`

	// ClonePath is the path to the polecat's clone of the rig.
	ClonePath string `json:"clone_path"`

	// Branch is the current git branch.
	Branch string `json:"branch"`

	// Issue is the currently assigned issue ID (if any).
	Issue string `json:"issue,omitempty"`

	// CreatedAt is when the polecat was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the polecat was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// Summary provides a concise view of polecat status.
type Summary struct {
	Name  string `json:"name"`
	State State  `json:"state"`
	Issue string `json:"issue,omitempty"`
}

// Summary returns a Summary for this polecat.
func (p *Polecat) Summary() Summary {
	return Summary{
		Name:  p.Name,
		State: p.State,
		Issue: p.Issue,
	}
}
