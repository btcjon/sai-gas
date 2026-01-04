package hooks

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HookRunner executes hooks for events.
type HookRunner struct {
	config   *HookConfig
	townRoot string
}

// NewRunner creates a new HookRunner with the given configuration.
func NewRunner(config *HookConfig, townRoot string) *HookRunner {
	return &HookRunner{
		config:   config,
		townRoot: townRoot,
	}
}

// NewRunnerFromTownRoot creates a HookRunner by loading config from the town root.
func NewRunnerFromTownRoot(townRoot string) (*HookRunner, error) {
	config, err := LoadConfig(townRoot)
	if err != nil {
		return nil, err
	}
	return NewRunner(config, townRoot), nil
}

// Fire executes all hooks registered for the given event.
// Returns results for each hook executed.
// For pre-* events, if any hook sets Block=true, subsequent hooks are not run.
func (r *HookRunner) Fire(ctx *EventContext) []HookResult {
	if r.config == nil {
		return nil
	}

	hooks := r.config.GetHooks(ctx.Event)
	if len(hooks) == 0 {
		return nil
	}

	results := make([]HookResult, 0, len(hooks))
	isPreEvent := ctx.Event.IsPreEvent()

	for _, hook := range hooks {
		result := r.runHook(ctx, hook)
		results = append(results, result)

		// For pre-* events, stop if a hook blocks
		if isPreEvent && result.Block {
			break
		}
	}

	return results
}

// FireEvent is a convenience method that creates context and fires.
func (r *HookRunner) FireEvent(event Event) []HookResult {
	ctx := NewEventContext(event, r.townRoot)
	return r.Fire(ctx)
}

// FireWithExtra fires an event with additional context.
func (r *HookRunner) FireWithExtra(event Event, extra map[string]string) []HookResult {
	ctx := NewEventContext(event, r.townRoot)
	ctx.Extra = extra
	return r.Fire(ctx)
}

// runHook executes a single hook and returns the result.
func (r *HookRunner) runHook(ctx *EventContext, hook Hook) HookResult {
	start := time.Now()

	result := HookResult{
		Hook: hook,
	}

	// Create timeout context
	timeout := hook.GetTimeout()
	execCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Parse command
	cmd := r.createCommand(execCtx, hook.Cmd)

	// Set up environment
	cmd.Env = append(os.Environ(), ctx.ToEnv()...)

	// Set working directory to town root
	cmd.Dir = r.townRoot

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	result.Elapsed = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = err.Error()

		// Check for timeout
		if execCtx.Err() == context.DeadlineExceeded {
			result.Error = "timeout after " + timeout.String()
		}

		// For pre-* events, check if the hook wants to block
		// Convention: exit code 2 means "block"
		if ctx.Event.IsPreEvent() {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 2 {
					result.Block = true
					result.Message = "hook requested block"
				}
			}
		}
	} else {
		result.Success = true
	}

	// Capture output (combined, truncated if needed)
	output := stdout.String() + stderr.String()
	if len(output) > 4096 {
		output = output[:4096] + "...(truncated)"
	}
	result.Output = strings.TrimSpace(output)

	// Parse message from output if present
	// Convention: last line starting with "MSG:" is the message
	lines := strings.Split(result.Output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "MSG:") {
			result.Message = strings.TrimPrefix(lines[i], "MSG:")
			result.Message = strings.TrimSpace(result.Message)
			break
		}
	}

	return result
}

// createCommand creates an exec.Cmd for the given command string.
// Uses shell execution for proper parsing of pipes, redirects, etc.
func (r *HookRunner) createCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", cmdStr)
}

// HasHooks returns true if any hooks are configured for the given event.
func (r *HookRunner) HasHooks(event Event) bool {
	return r.config != nil && r.config.HasHooks(event)
}

// Config returns the underlying hook configuration.
func (r *HookRunner) Config() *HookConfig {
	return r.config
}

// AnyBlocked returns true if any result has Block=true.
func AnyBlocked(results []HookResult) bool {
	for _, r := range results {
		if r.Block {
			return true
		}
	}
	return false
}

// AnyFailed returns true if any result has Success=false.
func AnyFailed(results []HookResult) bool {
	for _, r := range results {
		if !r.Success {
			return true
		}
	}
	return false
}

// AllSucceeded returns true if all results have Success=true.
func AllSucceeded(results []HookResult) bool {
	if len(results) == 0 {
		return true
	}
	return !AnyFailed(results)
}
