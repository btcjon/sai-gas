package monitoring

import (
	"regexp"
	"strings"
	"time"
)

// PatternMatch represents a matched status pattern in pane output.
type PatternMatch struct {
	Pattern string
	Status  string
}

// Pattern registry for status detection
var patterns = []struct {
	Pattern string
	Status  string
	Regex   *regexp.Regexp
}{
	{
		Pattern: "Thinking",
		Status:  StatusThinking,
		Regex:   regexp.MustCompile(`(?i)thinking\.{0,3}`),
	},
	{
		Pattern: "BLOCKED",
		Status:  StatusBlocked,
		Regex:   regexp.MustCompile(`(?i)blocked\s*:`),
	},
	{
		Pattern: "Error",
		Status:  StatusError,
		Regex:   regexp.MustCompile(`(?i)error\s*:`),
	},
	{
		Pattern: "Waiting",
		Status:  StatusBlocked,
		Regex:   regexp.MustCompile(`(?i)waiting\s+for`),
	},
	{
		Pattern: "Running",
		Status:  StatusWorking,
		Regex:   regexp.MustCompile(`(?i)running|processing`),
	},
	{
		Pattern: "Compiling",
		Status:  StatusWorking,
		Regex:   regexp.MustCompile(`(?i)compiling|building`),
	},
	{
		Pattern: "Failed",
		Status:  StatusError,
		Regex:   regexp.MustCompile(`(?i)failed|failure`),
	},
	{
		Pattern: "Panic",
		Status:  StatusError,
		Regex:   regexp.MustCompile(`(?i)panic|fatal`),
	},
}

// PatternDetector detects status patterns in agent pane output.
type PatternDetector struct {
	lastActivityTime map[string]time.Time // Track last activity per agent
}

// NewPatternDetector creates a new pattern detector.
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		lastActivityTime: make(map[string]time.Time),
	}
}

// DetectStatus analyzes pane output and infers agent status.
// Returns the detected status and the pattern that matched.
func (pd *PatternDetector) DetectStatus(address string, paneOutput string) (status string, pattern string) {
	if paneOutput == "" {
		return "", ""
	}

	// Trim whitespace and convert to lowercase for matching
	output := strings.TrimSpace(paneOutput)
	if output == "" {
		return "", ""
	}

	// Check patterns in order (higher priority first)
	for _, p := range patterns {
		if p.Regex.MatchString(output) {
			pd.lastActivityTime[address] = time.Now()
			return p.Status, p.Pattern
		}
	}

	// If pane has output but no pattern matched, agent is working
	pd.lastActivityTime[address] = time.Now()
	return StatusWorking, ""
}

// GetIdleStatus returns the idle duration for an agent based on last activity.
// If the agent has been idle longer than IdleThreshold, returns StatusIdle.
func (pd *PatternDetector) GetIdleStatus(address string, now time.Time) (status string, idleDuration time.Duration) {
	lastActivity, exists := pd.lastActivityTime[address]
	if !exists {
		// No recorded activity - unknown
		return "", 0
	}

	idleDuration = now.Sub(lastActivity)
	if idleDuration >= IdleThreshold {
		return StatusIdle, idleDuration
	}

	return "", 0
}

// RecordActivity updates the last activity time for an agent.
func (pd *PatternDetector) RecordActivity(address string) {
	pd.lastActivityTime[address] = time.Now()
}

// ResetActivity clears the activity record for an agent.
func (pd *PatternDetector) ResetActivity(address string) {
	delete(pd.lastActivityTime, address)
}

// LastActivity returns the last recorded activity time for an agent.
func (pd *PatternDetector) LastActivity(address string) (time.Time, bool) {
	t, exists := pd.lastActivityTime[address]
	return t, exists
}
