package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFile is the standard location for hooks configuration.
const ConfigFile = ".gastown/hooks.json"

var (
	// ErrConfigNotFound indicates the hooks config file does not exist.
	ErrConfigNotFound = errors.New("hooks config not found")

	// ErrInvalidConfig indicates the config file is malformed.
	ErrInvalidConfig = errors.New("invalid hooks config")
)

// LoadConfig loads the hooks configuration from the standard location.
// If the file doesn't exist, returns an empty config (not an error).
func LoadConfig(townRoot string) (*HookConfig, error) {
	path := filepath.Join(townRoot, ConfigFile)
	return LoadConfigFromPath(path)
}

// LoadConfigFromPath loads the hooks configuration from a specific path.
// If the file doesn't exist, returns an empty config (not an error).
func LoadConfigFromPath(path string) (*HookConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file = empty config (valid state)
			return &HookConfig{Hooks: make(map[Event][]Hook)}, nil
		}
		return nil, fmt.Errorf("reading hooks config: %w", err)
	}

	var config HookConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Initialize empty hooks map if nil
	if config.Hooks == nil {
		config.Hooks = make(map[Event][]Hook)
	}

	return &config, nil
}

// SaveConfig saves the hooks configuration to the standard location.
func SaveConfig(townRoot string, config *HookConfig) error {
	path := filepath.Join(townRoot, ConfigFile)
	return SaveConfigToPath(path, config)
}

// SaveConfigToPath saves the hooks configuration to a specific path.
func SaveConfigToPath(path string, config *HookConfig) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding hooks config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing hooks config: %w", err)
	}

	return nil
}

// ConfigExists checks if a hooks configuration file exists.
func ConfigExists(townRoot string) bool {
	path := filepath.Join(townRoot, ConfigFile)
	_, err := os.Stat(path)
	return err == nil
}

// ValidateConfig checks that a hooks configuration is valid.
func ValidateConfig(config *HookConfig) error {
	if config == nil {
		return fmt.Errorf("%w: nil config", ErrInvalidConfig)
	}

	for event, hooks := range config.Hooks {
		// Validate event name
		if !isValidEvent(event) {
			return fmt.Errorf("%w: unknown event type %q", ErrInvalidConfig, event)
		}

		// Validate each hook
		for i, hook := range hooks {
			if err := validateHook(event, i, hook); err != nil {
				return err
			}
		}
	}

	return nil
}

// isValidEvent checks if an event type is valid.
func isValidEvent(event Event) bool {
	for _, e := range AllEvents() {
		if e == event {
			return true
		}
	}
	return false
}

// validateHook validates a single hook definition.
func validateHook(event Event, index int, hook Hook) error {
	if hook.Type == "" {
		return fmt.Errorf("%w: hook %d for %q missing type", ErrInvalidConfig, index, event)
	}

	if hook.Type != HookTypeCommand {
		return fmt.Errorf("%w: hook %d for %q has unknown type %q", ErrInvalidConfig, index, event, hook.Type)
	}

	if hook.Cmd == "" {
		return fmt.Errorf("%w: hook %d for %q missing cmd", ErrInvalidConfig, index, event)
	}

	// Validate timeout if specified
	if hook.Timeout != "" {
		if hook.GetTimeout() <= 0 {
			return fmt.Errorf("%w: hook %d for %q has invalid timeout %q", ErrInvalidConfig, index, event, hook.Timeout)
		}
	}

	return nil
}

// NewEmptyConfig creates a new empty hooks configuration.
func NewEmptyConfig() *HookConfig {
	return &HookConfig{
		Hooks: make(map[Event][]Hook),
	}
}

// ExampleConfig returns an example hooks configuration for documentation.
func ExampleConfig() *HookConfig {
	return &HookConfig{
		Hooks: map[Event][]Hook{
			EventPreShutdown: {
				{Type: HookTypeCommand, Cmd: "./scripts/pre-shutdown.sh"},
			},
			EventPostSessionStart: {
				{Type: HookTypeCommand, Cmd: "echo 'Session started'"},
			},
			EventMailReceived: {
				{Type: HookTypeCommand, Cmd: "./scripts/on-mail.sh", Timeout: "10s"},
			},
		},
	}
}
