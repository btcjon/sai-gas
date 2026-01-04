package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAllEvents(t *testing.T) {
	events := AllEvents()
	if len(events) != 6 {
		t.Errorf("expected 6 events, got %d", len(events))
	}

	// Check all events are present
	expected := map[Event]bool{
		EventPreSessionStart:  true,
		EventPostSessionStart: true,
		EventPreShutdown:      true,
		EventPostShutdown:     true,
		EventMailReceived:     true,
		EventWorkAssigned:     true,
	}

	for _, e := range events {
		if !expected[e] {
			t.Errorf("unexpected event: %s", e)
		}
		delete(expected, e)
	}

	if len(expected) > 0 {
		t.Errorf("missing events: %v", expected)
	}
}

func TestEventIsPreEvent(t *testing.T) {
	tests := []struct {
		event    Event
		expected bool
	}{
		{EventPreSessionStart, true},
		{EventPreShutdown, true},
		{EventPostSessionStart, false},
		{EventPostShutdown, false},
		{EventMailReceived, false},
		{EventWorkAssigned, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			if got := tt.event.IsPreEvent(); got != tt.expected {
				t.Errorf("IsPreEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHookGetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
	}{
		{"empty", "", 30 * time.Second},
		{"valid", "10s", 10 * time.Second},
		{"invalid", "not-a-duration", 30 * time.Second},
		{"one minute", "1m", time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Hook{Timeout: tt.timeout}
			if got := h.GetTimeout(); got != tt.expected {
				t.Errorf("GetTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHookConfigGetHooks(t *testing.T) {
	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "echo shutdown"},
			},
		},
	}

	// Test getting existing hooks
	hooks := config.GetHooks(EventPreShutdown)
	if len(hooks) != 1 {
		t.Errorf("expected 1 hook, got %d", len(hooks))
	}

	// Test getting non-existent hooks
	hooks = config.GetHooks(EventMailReceived)
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks, got %d", len(hooks))
	}

	// Test nil config
	var nilConfig *HookConfig
	hooks = nilConfig.GetHooks(EventPreShutdown)
	if hooks != nil {
		t.Errorf("expected nil, got %v", hooks)
	}
}

func TestHookConfigHasHooks(t *testing.T) {
	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "echo shutdown"},
			},
		},
	}

	if !config.HasHooks(EventPreShutdown) {
		t.Error("expected HasHooks to return true for EventPreShutdown")
	}

	if config.HasHooks(EventMailReceived) {
		t.Error("expected HasHooks to return false for EventMailReceived")
	}
}

func TestEventContext(t *testing.T) {
	ctx := NewEventContext(EventPreShutdown, "/town/root").
		WithActor("gastown/polecats/nux").
		WithRig("gastown").
		WithExtra("MAIL_ID", "mail-123")

	if ctx.Event != EventPreShutdown {
		t.Errorf("expected event %s, got %s", EventPreShutdown, ctx.Event)
	}
	if ctx.TownRoot != "/town/root" {
		t.Errorf("expected town root '/town/root', got %s", ctx.TownRoot)
	}
	if ctx.Actor != "gastown/polecats/nux" {
		t.Errorf("expected actor 'gastown/polecats/nux', got %s", ctx.Actor)
	}
	if ctx.Rig != "gastown" {
		t.Errorf("expected rig 'gastown', got %s", ctx.Rig)
	}
	if ctx.Extra["MAIL_ID"] != "mail-123" {
		t.Errorf("expected extra MAIL_ID 'mail-123', got %s", ctx.Extra["MAIL_ID"])
	}
}

func TestEventContextToEnv(t *testing.T) {
	ctx := NewEventContext(EventPreShutdown, "/town/root").
		WithActor("test-actor").
		WithRig("test-rig")

	env := ctx.ToEnv()

	// Check required env vars are present
	hasEvent := false
	hasRoot := false
	hasActor := false
	hasRig := false

	for _, e := range env {
		switch {
		case e == "GT_HOOK_EVENT=pre-shutdown":
			hasEvent = true
		case e == "GT_TOWN_ROOT=/town/root":
			hasRoot = true
		case e == "GT_HOOK_ACTOR=test-actor":
			hasActor = true
		case e == "GT_HOOK_RIG=test-rig":
			hasRig = true
		}
	}

	if !hasEvent {
		t.Error("missing GT_HOOK_EVENT")
	}
	if !hasRoot {
		t.Error("missing GT_TOWN_ROOT")
	}
	if !hasActor {
		t.Error("missing GT_HOOK_ACTOR")
	}
	if !hasRig {
		t.Error("missing GT_HOOK_RIG")
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Test loading from non-existent file (should return empty config)
	config, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if len(config.Hooks) != 0 {
		t.Errorf("expected empty hooks, got %d", len(config.Hooks))
	}

	// Create a config file
	configDir := filepath.Join(tmpDir, ".gastown")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	testConfig := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "./scripts/pre-shutdown.sh"},
			},
			EventMailReceived: {
				{Type: HookTypeCommand, Cmd: "echo mail", Timeout: "10s"},
			},
		},
	}

	data, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load and verify
	config, err = LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(config.Hooks) != 2 {
		t.Errorf("expected 2 event types, got %d", len(config.Hooks))
	}

	shutdownHooks := config.GetHooks(EventPreShutdown)
	if len(shutdownHooks) != 1 {
		t.Errorf("expected 1 shutdown hook, got %d", len(shutdownHooks))
	}
	if shutdownHooks[0].Cmd != "./scripts/pre-shutdown.sh" {
		t.Errorf("unexpected cmd: %s", shutdownHooks[0].Cmd)
	}
}

func TestLoadConfigInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".gastown")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	_, err := LoadConfig(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "echo test"},
			},
		},
	}

	if err := SaveConfig(tmpDir, config); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, ".gastown", "hooks.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Reload and verify
	loaded, err := LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if !loaded.HasHooks(EventPreShutdown) {
		t.Error("expected loaded config to have shutdown hooks")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *HookConfig
		shouldErr bool
	}{
		{
			name:      "nil config",
			config:    nil,
			shouldErr: true,
		},
		{
			name: "valid config",
			config: &HookConfig{
				Hooks: map[Event][]Hook{
					EventPreShutdown: {
						{Type: HookTypeCommand, Cmd: "echo test"},
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "unknown event",
			config: &HookConfig{
				Hooks: map[Event][]Hook{
					"unknown-event": {
						{Type: HookTypeCommand, Cmd: "echo test"},
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "missing type",
			config: &HookConfig{
				Hooks: map[Event][]Hook{
					EventPreShutdown: {
						{Cmd: "echo test"},
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "unknown type",
			config: &HookConfig{
				Hooks: map[Event][]Hook{
					EventPreShutdown: {
						{Type: "webhook", Cmd: "echo test"},
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "missing cmd",
			config: &HookConfig{
				Hooks: map[Event][]Hook{
					EventPreShutdown: {
						{Type: HookTypeCommand},
					},
				},
			},
			shouldErr: true,
		},
		{
			name: "empty config",
			config: &HookConfig{
				Hooks: map[Event][]Hook{},
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if ConfigExists(tmpDir) {
		t.Error("expected ConfigExists to return false")
	}

	// Create the config
	configDir := filepath.Join(tmpDir, ".gastown")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "hooks.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Should exist now
	if !ConfigExists(tmpDir) {
		t.Error("expected ConfigExists to return true")
	}
}

func TestExampleConfig(t *testing.T) {
	example := ExampleConfig()
	if example == nil {
		t.Fatal("expected non-nil example config")
	}
	if len(example.Hooks) == 0 {
		t.Error("expected example to have hooks")
	}

	// Validate the example
	if err := ValidateConfig(example); err != nil {
		t.Errorf("example config is invalid: %v", err)
	}
}

func TestHookRunner(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "echo 'hello world'"},
			},
		},
	}

	runner := NewRunner(config, tmpDir)

	if !runner.HasHooks(EventPreShutdown) {
		t.Error("expected HasHooks to return true")
	}
	if runner.HasHooks(EventMailReceived) {
		t.Error("expected HasHooks to return false for unregistered event")
	}

	// Fire the event
	results := runner.FireEvent(EventPreShutdown)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.Output != "hello world" {
		t.Errorf("expected output 'hello world', got %q", result.Output)
	}
}

func TestHookRunnerWithMessage(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "echo 'MSG: custom message'"},
			},
		},
	}

	runner := NewRunner(config, tmpDir)
	results := runner.FireEvent(EventPreShutdown)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Message != "custom message" {
		t.Errorf("expected message 'custom message', got %q", results[0].Message)
	}
}

func TestHookRunnerFailure(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "exit 1"},
			},
		},
	}

	runner := NewRunner(config, tmpDir)
	results := runner.FireEvent(EventPreShutdown)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("expected failure")
	}
}

func TestHookRunnerBlock(t *testing.T) {
	tmpDir := t.TempDir()

	config := &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "exit 2"}, // Exit code 2 = block
				{Type: HookTypeCommand, Cmd: "echo second"}, // Should not run
			},
		},
	}

	runner := NewRunner(config, tmpDir)
	results := runner.Fire(NewEventContext(EventPreShutdown, tmpDir))

	// Only the first hook should have run (blocking)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (blocked), got %d", len(results))
	}

	if !results[0].Block {
		t.Error("expected first hook to block")
	}
}

func TestAnyBlocked(t *testing.T) {
	tests := []struct {
		name     string
		results  []HookResult
		expected bool
	}{
		{"empty", nil, false},
		{"none blocked", []HookResult{{Success: true}, {Success: false}}, false},
		{"one blocked", []HookResult{{Success: true}, {Block: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AnyBlocked(tt.results); got != tt.expected {
				t.Errorf("AnyBlocked() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAllSucceeded(t *testing.T) {
	tests := []struct {
		name     string
		results  []HookResult
		expected bool
	}{
		{"empty", nil, true},
		{"all success", []HookResult{{Success: true}, {Success: true}}, true},
		{"one failed", []HookResult{{Success: true}, {Success: false}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllSucceeded(tt.results); got != tt.expected {
				t.Errorf("AllSucceeded() = %v, want %v", got, tt.expected)
			}
		})
	}
}
