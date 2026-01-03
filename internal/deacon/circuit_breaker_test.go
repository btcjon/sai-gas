package deacon

import (
	"os"
	"testing"
	"time"
)

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()

	if cfg.FailureThreshold != DefaultFailureThreshold {
		t.Errorf("FailureThreshold = %d, want %d", cfg.FailureThreshold, DefaultFailureThreshold)
	}
	if cfg.RecoveryTimeout != DefaultRecoveryTimeout {
		t.Errorf("RecoveryTimeout = %v, want %v", cfg.RecoveryTimeout, DefaultRecoveryTimeout)
	}
	if cfg.SuccessThreshold != DefaultSuccessThreshold {
		t.Errorf("SuccessThreshold = %d, want %d", cfg.SuccessThreshold, DefaultSuccessThreshold)
	}
	if cfg.MaxBackoff != DefaultMaxBackoff {
		t.Errorf("MaxBackoff = %v, want %v", cfg.MaxBackoff, DefaultMaxBackoff)
	}
	if cfg.BackoffMultiplier != DefaultBackoffMultiplier {
		t.Errorf("BackoffMultiplier = %f, want %f", cfg.BackoffMultiplier, DefaultBackoffMultiplier)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-agent")

	if cb.ID != "test-agent" {
		t.Errorf("ID = %q, want %q", cb.ID, "test-agent")
	}
	if cb.State != CircuitClosed {
		t.Errorf("State = %v, want %v", cb.State, CircuitClosed)
	}
	if cb.Failures != 0 {
		t.Errorf("Failures = %d, want 0", cb.Failures)
	}
	if cb.LastStateChange.IsZero() {
		t.Error("LastStateChange should be set")
	}
}

func TestCircuitBreaker_RecordSuccess_Closed(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	changed := cb.RecordSuccess(cfg)

	if changed {
		t.Error("RecordSuccess should not change state when already closed")
	}
	if cb.State != CircuitClosed {
		t.Errorf("State = %v, want %v", cb.State, CircuitClosed)
	}
	if cb.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses = %d, want 1", cb.TotalSuccesses)
	}
}

func TestCircuitBreaker_RecordFailure_TripsOpen(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// Record failures up to threshold
	for i := 0; i < cfg.FailureThreshold-1; i++ {
		changed := cb.RecordFailure(cfg)
		if changed {
			t.Errorf("Failure %d should not trip circuit", i+1)
		}
		if cb.State != CircuitClosed {
			t.Errorf("State after %d failures = %v, want %v", i+1, cb.State, CircuitClosed)
		}
	}

	// This should trip the circuit
	changed := cb.RecordFailure(cfg)
	if !changed {
		t.Error("Failure at threshold should trip circuit")
	}
	if cb.State != CircuitOpen {
		t.Errorf("State = %v, want %v", cb.State, CircuitOpen)
	}
	if cb.TripCount != 1 {
		t.Errorf("TripCount = %d, want 1", cb.TripCount)
	}
	if cb.TotalFailures != int64(cfg.FailureThreshold) {
		t.Errorf("TotalFailures = %d, want %d", cb.TotalFailures, cfg.FailureThreshold)
	}
}

func TestCircuitBreaker_ShouldAllow_Closed(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	if !cb.ShouldAllow(cfg) {
		t.Error("Closed circuit should allow operations")
	}
}

func TestCircuitBreaker_ShouldAllow_Open(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// Trip the circuit
	for i := 0; i < cfg.FailureThreshold; i++ {
		cb.RecordFailure(cfg)
	}

	// Should not allow immediately
	if cb.ShouldAllow(cfg) {
		t.Error("Open circuit should not allow operations immediately")
	}

	// Simulate time passing
	cb.LastStateChange = time.Now().Add(-cfg.RecoveryTimeout - time.Second)

	// Should now allow (and transition to half-open)
	if !cb.ShouldAllow(cfg) {
		t.Error("Open circuit should allow after recovery timeout")
	}
	if cb.State != CircuitHalfOpen {
		t.Errorf("State = %v, want %v", cb.State, CircuitHalfOpen)
	}
}

func TestCircuitBreaker_HalfOpenToClose(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// Trip the circuit
	for i := 0; i < cfg.FailureThreshold; i++ {
		cb.RecordFailure(cfg)
	}

	// Transition to half-open
	cb.LastStateChange = time.Now().Add(-cfg.RecoveryTimeout - time.Second)
	cb.ShouldAllow(cfg)

	if cb.State != CircuitHalfOpen {
		t.Fatalf("Expected half-open state, got %v", cb.State)
	}

	// Record successes
	for i := 0; i < cfg.SuccessThreshold-1; i++ {
		changed := cb.RecordSuccess(cfg)
		if changed {
			t.Errorf("Success %d should not close circuit yet", i+1)
		}
	}

	// Final success should close
	changed := cb.RecordSuccess(cfg)
	if !changed {
		t.Error("Final success should close circuit")
	}
	if cb.State != CircuitClosed {
		t.Errorf("State = %v, want %v", cb.State, CircuitClosed)
	}
	if cb.CurrentBackoff != 0 {
		t.Errorf("CurrentBackoff should reset to 0, got %v", cb.CurrentBackoff)
	}
}

func TestCircuitBreaker_HalfOpenFailure(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// Trip the circuit
	for i := 0; i < cfg.FailureThreshold; i++ {
		cb.RecordFailure(cfg)
	}

	// Transition to half-open
	cb.LastStateChange = time.Now().Add(-cfg.RecoveryTimeout - time.Second)
	cb.ShouldAllow(cfg)

	if cb.State != CircuitHalfOpen {
		t.Fatalf("Expected half-open state, got %v", cb.State)
	}

	initialTripCount := cb.TripCount

	// Failure during half-open should reopen
	changed := cb.RecordFailure(cfg)
	if !changed {
		t.Error("Failure during half-open should trip circuit")
	}
	if cb.State != CircuitOpen {
		t.Errorf("State = %v, want %v", cb.State, CircuitOpen)
	}
	if cb.TripCount != initialTripCount+1 {
		t.Errorf("TripCount = %d, want %d", cb.TripCount, initialTripCount+1)
	}
}

func TestCircuitBreaker_ExponentialBackoff(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := &CircuitBreakerConfig{
		FailureThreshold:  1,
		RecoveryTimeout:   10 * time.Second,
		SuccessThreshold:  1,
		MaxBackoff:        5 * time.Minute,
		BackoffMultiplier: 2.0,
	}

	// First trip: base timeout
	cb.RecordFailure(cfg)
	if cb.CurrentBackoff != 10*time.Second {
		t.Errorf("First trip backoff = %v, want 10s", cb.CurrentBackoff)
	}

	// Recover
	cb.LastStateChange = time.Now().Add(-15 * time.Second)
	cb.ShouldAllow(cfg)
	cb.RecordSuccess(cfg)

	// Second trip: 2x base
	cb.RecordFailure(cfg)
	if cb.CurrentBackoff != 20*time.Second {
		t.Errorf("Second trip backoff = %v, want 20s", cb.CurrentBackoff)
	}

	// Recover again
	cb.LastStateChange = time.Now().Add(-25 * time.Second)
	cb.ShouldAllow(cfg)
	cb.RecordSuccess(cfg)

	// Third trip: 4x base
	cb.RecordFailure(cfg)
	if cb.CurrentBackoff != 40*time.Second {
		t.Errorf("Third trip backoff = %v, want 40s", cb.CurrentBackoff)
	}
}

func TestCircuitBreaker_BackoffMaxCap(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := &CircuitBreakerConfig{
		FailureThreshold:  1,
		RecoveryTimeout:   10 * time.Second,
		SuccessThreshold:  1,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Simulate many trips
	cb.TripCount = 10

	cb.RecordFailure(cfg)

	// Backoff should be capped at max
	if cb.CurrentBackoff > cfg.MaxBackoff {
		t.Errorf("Backoff = %v, should not exceed %v", cb.CurrentBackoff, cfg.MaxBackoff)
	}
	if cb.CurrentBackoff != cfg.MaxBackoff {
		t.Errorf("Backoff = %v, want %v (max)", cb.CurrentBackoff, cfg.MaxBackoff)
	}
}

func TestCircuitBreaker_TimeUntilRetry(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// Closed circuit
	if cb.TimeUntilRetry(cfg) != 0 {
		t.Error("Closed circuit should have 0 time until retry")
	}

	// Trip the circuit
	for i := 0; i < cfg.FailureThreshold; i++ {
		cb.RecordFailure(cfg)
	}

	// Should have time remaining
	remaining := cb.TimeUntilRetry(cfg)
	if remaining <= 0 {
		t.Error("Open circuit should have positive time until retry")
	}
	if remaining > cfg.RecoveryTimeout {
		t.Errorf("TimeUntilRetry = %v, should not exceed %v", remaining, cfg.RecoveryTimeout)
	}
}

func TestCircuitBreaker_SuccessRate(t *testing.T) {
	cb := NewCircuitBreaker("test")
	cfg := DefaultCircuitBreakerConfig()

	// No operations
	if cb.SuccessRate() != 100.0 {
		t.Errorf("Empty circuit success rate = %f, want 100.0", cb.SuccessRate())
	}

	// 2 successes, 1 failure = 66.67%
	cb.RecordSuccess(cfg)
	cb.RecordSuccess(cfg)
	cb.RecordFailure(cfg)

	expected := 200.0 / 3.0
	if rate := cb.SuccessRate(); rate < expected-0.1 || rate > expected+0.1 {
		t.Errorf("Success rate = %f, want ~%f", rate, expected)
	}
}

func TestCircuitBreakerRegistryFile(t *testing.T) {
	path := CircuitBreakerRegistryFile("/tmp/test-town")
	expected := "/tmp/test-town/deacon/circuit-breakers.json"
	if path != expected {
		t.Errorf("CircuitBreakerRegistryFile = %q, want %q", path, expected)
	}
}

func TestLoadCircuitBreakerRegistry_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	reg, err := LoadCircuitBreakerRegistry(tmpDir)
	if err != nil {
		t.Fatalf("LoadCircuitBreakerRegistry() error = %v", err)
	}
	if reg.Breakers == nil {
		t.Error("Breakers map should be initialized")
	}
	if len(reg.Breakers) != 0 {
		t.Errorf("Expected empty breakers map, got %d entries", len(reg.Breakers))
	}
	if reg.Config == nil {
		t.Error("Config should be initialized")
	}
}

func TestSaveAndLoadCircuitBreakerRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry with data
	reg := &CircuitBreakerRegistry{
		Breakers: map[string]*CircuitBreaker{
			"agent-1": {
				ID:             "agent-1",
				State:          CircuitOpen,
				TripCount:      3,
				TotalSuccesses: 10,
				TotalFailures:  5,
			},
		},
		Config: DefaultCircuitBreakerConfig(),
	}

	// Save
	if err := SaveCircuitBreakerRegistry(tmpDir, reg); err != nil {
		t.Fatalf("SaveCircuitBreakerRegistry() error = %v", err)
	}

	// Verify file exists
	regFile := CircuitBreakerRegistryFile(tmpDir)
	if _, err := os.Stat(regFile); os.IsNotExist(err) {
		t.Fatal("Registry file was not created")
	}

	// Load
	loaded, err := LoadCircuitBreakerRegistry(tmpDir)
	if err != nil {
		t.Fatalf("LoadCircuitBreakerRegistry() error = %v", err)
	}

	// Verify loaded data
	cb := loaded.Breakers["agent-1"]
	if cb == nil {
		t.Fatal("Breaker not found in loaded registry")
	}
	if cb.State != CircuitOpen {
		t.Errorf("State = %v, want %v", cb.State, CircuitOpen)
	}
	if cb.TripCount != 3 {
		t.Errorf("TripCount = %d, want 3", cb.TripCount)
	}
}

func TestCircuitBreakerRegistry_GetBreaker(t *testing.T) {
	reg := &CircuitBreakerRegistry{}

	// First call creates the breaker
	cb1 := reg.GetBreaker("test-agent")
	if cb1 == nil {
		t.Fatal("GetBreaker returned nil")
	}
	if cb1.ID != "test-agent" {
		t.Errorf("ID = %q, want %q", cb1.ID, "test-agent")
	}

	// Second call returns same breaker
	cb2 := reg.GetBreaker("test-agent")
	if cb1 != cb2 {
		t.Error("GetBreaker should return the same pointer")
	}
}

func TestCircuitBreakerRegistry_OpenBreakers(t *testing.T) {
	reg := &CircuitBreakerRegistry{
		Breakers: map[string]*CircuitBreaker{
			"closed":    {ID: "closed", State: CircuitClosed},
			"open-1":    {ID: "open-1", State: CircuitOpen},
			"half-open": {ID: "half-open", State: CircuitHalfOpen},
			"open-2":    {ID: "open-2", State: CircuitOpen},
		},
	}

	open := reg.OpenBreakers()
	if len(open) != 2 {
		t.Errorf("Expected 2 open breakers, got %d", len(open))
	}
}

func TestCircuitBreakerRegistry_HalfOpenBreakers(t *testing.T) {
	reg := &CircuitBreakerRegistry{
		Breakers: map[string]*CircuitBreaker{
			"closed":      {ID: "closed", State: CircuitClosed},
			"half-open-1": {ID: "half-open-1", State: CircuitHalfOpen},
			"open":        {ID: "open", State: CircuitOpen},
			"half-open-2": {ID: "half-open-2", State: CircuitHalfOpen},
		},
	}

	halfOpen := reg.HalfOpenBreakers()
	if len(halfOpen) != 2 {
		t.Errorf("Expected 2 half-open breakers, got %d", len(halfOpen))
	}
}

func TestCircuitBreakerRegistry_Summary(t *testing.T) {
	reg := &CircuitBreakerRegistry{
		Breakers: map[string]*CircuitBreaker{
			"c1": {State: CircuitClosed},
			"c2": {State: CircuitClosed},
			"o1": {State: CircuitOpen},
			"h1": {State: CircuitHalfOpen},
		},
	}

	summary := reg.Summary()
	if summary[CircuitClosed] != 2 {
		t.Errorf("Closed count = %d, want 2", summary[CircuitClosed])
	}
	if summary[CircuitOpen] != 1 {
		t.Errorf("Open count = %d, want 1", summary[CircuitOpen])
	}
	if summary[CircuitHalfOpen] != 1 {
		t.Errorf("HalfOpen count = %d, want 1", summary[CircuitHalfOpen])
	}
}

func TestCircuitBreakerRegistry_GetConfig(t *testing.T) {
	// With nil config
	reg := &CircuitBreakerRegistry{}
	cfg := reg.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig should return default config")
	}
	if cfg.FailureThreshold != DefaultFailureThreshold {
		t.Errorf("Default config not applied correctly")
	}

	// With existing config
	customCfg := &CircuitBreakerConfig{FailureThreshold: 10}
	reg.Config = customCfg
	cfg = reg.GetConfig()
	if cfg.FailureThreshold != 10 {
		t.Errorf("Custom config not returned")
	}
}
