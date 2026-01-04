// Package hooks provides an event hook system for Gas Town extensibility.
//
// Hooks allow external scripts to run in response to system events like
// session start/stop, mail received, and work assignment. They are configured
// via .gastown/hooks.json in the town root.
package hooks

import (
	"time"
)

// Event represents a hook event type.
type Event string

// Supported event types.
const (
	// Pre/post session events
	EventPreSessionStart  Event = "pre-session-start"
	EventPostSessionStart Event = "post-session-start"

	// Shutdown events
	EventPreShutdown  Event = "pre-shutdown"
	EventPostShutdown Event = "post-shutdown"

	// Communication events
	EventMailReceived Event = "mail-received"
	EventWorkAssigned Event = "work-assigned"
)

// AllEvents returns all supported event types.
func AllEvents() []Event {
	return []Event{
		EventPreSessionStart,
		EventPostSessionStart,
		EventPreShutdown,
		EventPostShutdown,
		EventMailReceived,
		EventWorkAssigned,
	}
}

// IsPreEvent returns true if this is a pre-* event that can block.
func (e Event) IsPreEvent() bool {
	switch e {
	case EventPreSessionStart, EventPreShutdown:
		return true
	default:
		return false
	}
}

// String returns the event name.
func (e Event) String() string {
	return string(e)
}

// HookType represents the type of hook action.
type HookType string

// Supported hook types.
const (
	HookTypeCommand HookType = "command"
)

// Hook represents a single hook definition.
type Hook struct {
	Type    HookType `json:"type"`              // "command"
	Cmd     string   `json:"cmd"`               // command to execute
	Timeout string   `json:"timeout,omitempty"` // optional timeout (e.g., "30s")
}

// GetTimeout returns the timeout duration, defaulting to 30s.
func (h *Hook) GetTimeout() time.Duration {
	if h.Timeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// HookConfig represents the hooks configuration file structure.
type HookConfig struct {
	Hooks map[Event][]Hook `json:"hooks"`
}

// GetHooks returns the hooks registered for a specific event.
func (c *HookConfig) GetHooks(event Event) []Hook {
	if c == nil || c.Hooks == nil {
		return nil
	}
	return c.Hooks[event]
}

// HasHooks returns true if there are any hooks for the given event.
func (c *HookConfig) HasHooks(event Event) bool {
	return len(c.GetHooks(event)) > 0
}

// AllRegisteredEvents returns all events that have at least one hook.
func (c *HookConfig) AllRegisteredEvents() []Event {
	if c == nil || c.Hooks == nil {
		return nil
	}
	var events []Event
	for e, hooks := range c.Hooks {
		if len(hooks) > 0 {
			events = append(events, e)
		}
	}
	return events
}

// HookResult represents the result of running a single hook.
type HookResult struct {
	Hook    Hook          `json:"hook"`
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Block   bool          `json:"block,omitempty"` // For pre-* hooks
	Output  string        `json:"output,omitempty"`
	Error   string        `json:"error,omitempty"`
	Elapsed time.Duration `json:"elapsed"`
}

// EventContext contains contextual information passed to hooks.
type EventContext struct {
	// Event is the event type being fired.
	Event Event `json:"event"`

	// TownRoot is the root directory of the Gas Town workspace.
	TownRoot string `json:"town_root"`

	// Actor is the agent firing the event (e.g., "gastown/polecats/nux").
	Actor string `json:"actor,omitempty"`

	// Rig is the rig name, if applicable.
	Rig string `json:"rig,omitempty"`

	// Extra contains event-specific key-value pairs.
	Extra map[string]string `json:"extra,omitempty"`
}

// NewEventContext creates a new EventContext with the given event and town root.
func NewEventContext(event Event, townRoot string) *EventContext {
	return &EventContext{
		Event:    event,
		TownRoot: townRoot,
		Extra:    make(map[string]string),
	}
}

// WithActor sets the actor and returns the context for chaining.
func (c *EventContext) WithActor(actor string) *EventContext {
	c.Actor = actor
	return c
}

// WithRig sets the rig and returns the context for chaining.
func (c *EventContext) WithRig(rig string) *EventContext {
	c.Rig = rig
	return c
}

// WithExtra adds extra key-value pairs and returns the context for chaining.
func (c *EventContext) WithExtra(key, value string) *EventContext {
	if c.Extra == nil {
		c.Extra = make(map[string]string)
	}
	c.Extra[key] = value
	return c
}

// ToEnv converts the context to environment variables.
func (c *EventContext) ToEnv() []string {
	env := []string{
		"GT_HOOK_EVENT=" + string(c.Event),
		"GT_TOWN_ROOT=" + c.TownRoot,
	}
	if c.Actor != "" {
		env = append(env, "GT_HOOK_ACTOR="+c.Actor)
	}
	if c.Rig != "" {
		env = append(env, "GT_HOOK_RIG="+c.Rig)
	}
	for k, v := range c.Extra {
		env = append(env, "GT_HOOK_"+k+"="+v)
	}
	return env
}
