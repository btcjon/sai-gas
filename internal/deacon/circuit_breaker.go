// Package deacon provides the Deacon agent infrastructure.
package deacon

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState string

const (
	// CircuitClosed means the circuit is functioning normally.
	// All operations are allowed through.
	CircuitClosed CircuitState = "closed"

	// CircuitOpen means the circuit has tripped due to failures.
	// Operations are blocked to allow the system to recover.
	CircuitOpen CircuitState = "open"

	// CircuitHalfOpen means the circuit is testing if recovery is possible.
	// A limited number of operations are allowed through.
	CircuitHalfOpen CircuitState = "half_open"
)

// Default circuit breaker parameters.
const (
	DefaultFailureThreshold  = 3               // Failures before opening
	DefaultRecoveryTimeout   = 30 * time.Second // Time before trying half-open
	DefaultSuccessThreshold  = 2               // Successes needed to close from half-open
	DefaultMaxBackoff        = 5 * time.Minute // Maximum backoff duration
	DefaultBackoffMultiplier = 2.0             // Exponential backoff multiplier
)

// CircuitBreakerConfig holds configurable parameters for a circuit breaker.
type CircuitBreakerConfig struct {
	FailureThreshold  int           `json:"failure_threshold"`
	RecoveryTimeout   time.Duration `json:"recovery_timeout"`
	SuccessThreshold  int           `json:"success_threshold"`
	MaxBackoff        time.Duration `json:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
}

// DefaultCircuitBreakerConfig returns the default circuit breaker config.
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold:  DefaultFailureThreshold,
		RecoveryTimeout:   DefaultRecoveryTimeout,
		SuccessThreshold:  DefaultSuccessThreshold,
		MaxBackoff:        DefaultMaxBackoff,
		BackoffMultiplier: DefaultBackoffMultiplier,
	}
}

// CircuitBreaker tracks state for a single agent or operation.
type CircuitBreaker struct {
	// ID identifies this circuit (e.g., agent ID or operation name)
	ID string `json:"id"`

	// State is the current circuit state
	State CircuitState `json:"state"`

	// Failures counts consecutive failures while closed
	Failures int `json:"failures"`

	// Successes counts consecutive successes while half-open
	Successes int `json:"successes"`

	// LastFailureTime is when the last failure occurred
	LastFailureTime time.Time `json:"last_failure_time,omitempty"`

	// LastStateChange is when the state last changed
	LastStateChange time.Time `json:"last_state_change"`

	// TripCount is how many times the circuit has tripped (closed -> open)
	TripCount int `json:"trip_count"`

	// CurrentBackoff is the current backoff duration (increases with repeated trips)
	CurrentBackoff time.Duration `json:"current_backoff"`

	// TotalSuccesses tracks lifetime successes for metrics
	TotalSuccesses int64 `json:"total_successes"`

	// TotalFailures tracks lifetime failures for metrics
	TotalFailures int64 `json:"total_failures"`
}

// NewCircuitBreaker creates a new circuit breaker in closed state.
func NewCircuitBreaker(id string) *CircuitBreaker {
	return &CircuitBreaker{
		ID:              id,
		State:           CircuitClosed,
		LastStateChange: time.Now().UTC(),
	}
}

// RecordSuccess records a successful operation.
// Returns true if the circuit state changed.
func (cb *CircuitBreaker) RecordSuccess(cfg *CircuitBreakerConfig) bool {
	cb.TotalSuccesses++
	cb.Failures = 0 // Reset failure count on success

	switch cb.State {
	case CircuitClosed:
		// Already closed, nothing to do
		return false

	case CircuitHalfOpen:
		cb.Successes++
		if cb.Successes >= cfg.SuccessThreshold {
			// Enough successes to close the circuit
			cb.transitionTo(CircuitClosed)
			cb.Successes = 0
			// Reset backoff on successful recovery
			cb.CurrentBackoff = 0
			return true
		}
		return false

	case CircuitOpen:
		// Shouldn't happen - operations blocked when open
		// But if it does, treat as recovery attempt
		cb.transitionTo(CircuitHalfOpen)
		cb.Successes = 1
		return true
	}

	return false
}

// RecordFailure records a failed operation.
// Returns true if the circuit state changed (opened).
func (cb *CircuitBreaker) RecordFailure(cfg *CircuitBreakerConfig) bool {
	cb.TotalFailures++
	cb.LastFailureTime = time.Now().UTC()

	switch cb.State {
	case CircuitClosed:
		cb.Failures++
		if cb.Failures >= cfg.FailureThreshold {
			cb.transitionTo(CircuitOpen)
			cb.TripCount++
			cb.calculateBackoff(cfg)
			return true
		}
		return false

	case CircuitHalfOpen:
		// Failed during recovery - back to open
		cb.transitionTo(CircuitOpen)
		cb.Successes = 0
		cb.TripCount++
		cb.calculateBackoff(cfg)
		return true

	case CircuitOpen:
		// Already open, update backoff
		cb.calculateBackoff(cfg)
		return false
	}

	return false
}

// transitionTo changes the circuit state and records the time.
func (cb *CircuitBreaker) transitionTo(state CircuitState) {
	cb.State = state
	cb.LastStateChange = time.Now().UTC()
}

// calculateBackoff calculates the current backoff using exponential backoff.
// The backoff increases with each trip: base * multiplier^(tripCount-1)
func (cb *CircuitBreaker) calculateBackoff(cfg *CircuitBreakerConfig) {
	if cb.TripCount <= 0 {
		cb.CurrentBackoff = cfg.RecoveryTimeout
		return
	}

	// Exponential backoff: base * multiplier^(trips-1)
	// Cap the exponent to avoid overflow
	exponent := float64(cb.TripCount - 1)
	if exponent > 10 {
		exponent = 10 // Cap at reasonable level
	}

	multiplier := math.Pow(cfg.BackoffMultiplier, exponent)
	backoff := time.Duration(float64(cfg.RecoveryTimeout) * multiplier)

	// Cap at max backoff
	if backoff > cfg.MaxBackoff {
		backoff = cfg.MaxBackoff
	}

	cb.CurrentBackoff = backoff
}

// ShouldAllow returns true if an operation should be allowed through.
// For open circuits, checks if enough time has passed to try half-open.
func (cb *CircuitBreaker) ShouldAllow(cfg *CircuitBreakerConfig) bool {
	switch cb.State {
	case CircuitClosed:
		return true

	case CircuitHalfOpen:
		// Allow limited operations during half-open
		return true

	case CircuitOpen:
		// Check if we should try half-open
		backoff := cb.CurrentBackoff
		if backoff == 0 {
			backoff = cfg.RecoveryTimeout
		}
		if time.Since(cb.LastStateChange) >= backoff {
			cb.transitionTo(CircuitHalfOpen)
			cb.Successes = 0
			return true
		}
		return false
	}

	return false
}

// TimeUntilRetry returns how long until a retry should be attempted.
// Returns 0 if the circuit allows operations.
func (cb *CircuitBreaker) TimeUntilRetry(cfg *CircuitBreakerConfig) time.Duration {
	if cb.State != CircuitOpen {
		return 0
	}

	backoff := cb.CurrentBackoff
	if backoff == 0 {
		backoff = cfg.RecoveryTimeout
	}

	elapsed := time.Since(cb.LastStateChange)
	if elapsed >= backoff {
		return 0
	}

	return backoff - elapsed
}

// SuccessRate returns the success rate as a percentage (0-100).
// Returns 100 if no operations have been recorded.
func (cb *CircuitBreaker) SuccessRate() float64 {
	total := cb.TotalSuccesses + cb.TotalFailures
	if total == 0 {
		return 100.0
	}
	return float64(cb.TotalSuccesses) / float64(total) * 100.0
}

// CircuitBreakerRegistry holds circuit breakers for all agents/operations.
type CircuitBreakerRegistry struct {
	// Breakers maps ID to circuit breaker
	Breakers map[string]*CircuitBreaker `json:"breakers"`

	// LastUpdated is when this registry was last written
	LastUpdated time.Time `json:"last_updated"`

	// Config is the shared configuration for all breakers
	Config *CircuitBreakerConfig `json:"config"`
}

// CircuitBreakerRegistryFile returns the path to the circuit breaker registry.
func CircuitBreakerRegistryFile(townRoot string) string {
	return filepath.Join(townRoot, "deacon", "circuit-breakers.json")
}

// LoadCircuitBreakerRegistry loads the circuit breaker registry from disk.
// Returns empty registry if file doesn't exist.
func LoadCircuitBreakerRegistry(townRoot string) (*CircuitBreakerRegistry, error) {
	regFile := CircuitBreakerRegistryFile(townRoot)

	data, err := os.ReadFile(regFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &CircuitBreakerRegistry{
				Breakers: make(map[string]*CircuitBreaker),
				Config:   DefaultCircuitBreakerConfig(),
			}, nil
		}
		return nil, fmt.Errorf("reading circuit breaker registry: %w", err)
	}

	var reg CircuitBreakerRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing circuit breaker registry: %w", err)
	}

	if reg.Breakers == nil {
		reg.Breakers = make(map[string]*CircuitBreaker)
	}
	if reg.Config == nil {
		reg.Config = DefaultCircuitBreakerConfig()
	}

	return &reg, nil
}

// SaveCircuitBreakerRegistry saves the circuit breaker registry to disk.
func SaveCircuitBreakerRegistry(townRoot string, reg *CircuitBreakerRegistry) error {
	regFile := CircuitBreakerRegistryFile(townRoot)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(regFile), 0755); err != nil {
		return fmt.Errorf("creating deacon directory: %w", err)
	}

	reg.LastUpdated = time.Now().UTC()

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling circuit breaker registry: %w", err)
	}

	return os.WriteFile(regFile, data, 0644)
}

// GetBreaker returns the circuit breaker for an ID, creating if needed.
func (reg *CircuitBreakerRegistry) GetBreaker(id string) *CircuitBreaker {
	if reg.Breakers == nil {
		reg.Breakers = make(map[string]*CircuitBreaker)
	}

	cb, ok := reg.Breakers[id]
	if !ok {
		cb = NewCircuitBreaker(id)
		reg.Breakers[id] = cb
	}
	return cb
}

// GetConfig returns the registry config, using defaults if not set.
func (reg *CircuitBreakerRegistry) GetConfig() *CircuitBreakerConfig {
	if reg.Config == nil {
		reg.Config = DefaultCircuitBreakerConfig()
	}
	return reg.Config
}

// OpenBreakers returns all circuit breakers that are currently open.
func (reg *CircuitBreakerRegistry) OpenBreakers() []*CircuitBreaker {
	var open []*CircuitBreaker
	for _, cb := range reg.Breakers {
		if cb.State == CircuitOpen {
			open = append(open, cb)
		}
	}
	return open
}

// HalfOpenBreakers returns all circuit breakers in half-open state.
func (reg *CircuitBreakerRegistry) HalfOpenBreakers() []*CircuitBreaker {
	var halfOpen []*CircuitBreaker
	for _, cb := range reg.Breakers {
		if cb.State == CircuitHalfOpen {
			halfOpen = append(halfOpen, cb)
		}
	}
	return halfOpen
}

// Summary returns a summary of the circuit breaker states.
func (reg *CircuitBreakerRegistry) Summary() map[CircuitState]int {
	summary := map[CircuitState]int{
		CircuitClosed:   0,
		CircuitOpen:     0,
		CircuitHalfOpen: 0,
	}
	for _, cb := range reg.Breakers {
		summary[cb.State]++
	}
	return summary
}
