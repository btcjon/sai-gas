// Package beads provides field parsing utilities for structured issue descriptions.
package beads

import (
	"fmt"
	"strings"
	"time"
)

// Note: AgentFields, ParseAgentFields, FormatAgentDescription, and CreateAgentBead are in beads.go

// ParseAgentFieldsFromDescription is an alias for ParseAgentFields.
// Used by daemon for compatibility.
func ParseAgentFieldsFromDescription(description string) *AgentFields {
	return ParseAgentFields(description)
}

// AttachmentFields holds the attachment info for pinned beads.
// These fields track which molecule is attached to a handoff/pinned bead.
type AttachmentFields struct {
	AttachedMolecule string // Root issue ID of the attached molecule
	AttachedAt       string // ISO 8601 timestamp when attached
	AttachedArgs     string // Natural language args passed via gt sling --args (no-tmux mode)
}

// ParseAttachmentFields extracts attachment fields from an issue's description.
// Fields are expected as "key: value" lines. Returns nil if no attachment fields found.
func ParseAttachmentFields(issue *Issue) *AttachmentFields {
	if issue == nil || issue.Description == "" {
		return nil
	}

	fields := &AttachmentFields{}
	hasFields := false

	for _, line := range strings.Split(issue.Description, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for "key: value" pattern
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		if value == "" {
			continue
		}

		// Map keys to fields (case-insensitive)
		switch strings.ToLower(key) {
		case "attached_molecule", "attached-molecule", "attachedmolecule":
			fields.AttachedMolecule = value
			hasFields = true
		case "attached_at", "attached-at", "attachedat":
			fields.AttachedAt = value
			hasFields = true
		case "attached_args", "attached-args", "attachedargs":
			fields.AttachedArgs = value
			hasFields = true
		}
	}

	if !hasFields {
		return nil
	}
	return fields
}

// FormatAttachmentFields formats AttachmentFields as a string suitable for an issue description.
// Only non-empty fields are included.
func FormatAttachmentFields(fields *AttachmentFields) string {
	if fields == nil {
		return ""
	}

	var lines []string

	if fields.AttachedMolecule != "" {
		lines = append(lines, "attached_molecule: "+fields.AttachedMolecule)
	}
	if fields.AttachedAt != "" {
		lines = append(lines, "attached_at: "+fields.AttachedAt)
	}
	if fields.AttachedArgs != "" {
		lines = append(lines, "attached_args: "+fields.AttachedArgs)
	}

	return strings.Join(lines, "\n")
}

// SetAttachmentFields updates an issue's description with the given attachment fields.
// Existing attachment field lines are replaced; other content is preserved.
// Returns the new description string.
func SetAttachmentFields(issue *Issue, fields *AttachmentFields) string {
	// Known attachment field keys (lowercase)
	attachmentKeys := map[string]bool{
		"attached_molecule": true,
		"attached-molecule": true,
		"attachedmolecule":  true,
		"attached_at":       true,
		"attached-at":       true,
		"attachedat":        true,
		"attached_args":     true,
		"attached-args":     true,
		"attachedargs":      true,
	}

	// Collect non-attachment lines from existing description
	var otherLines []string
	if issue != nil && issue.Description != "" {
		for _, line := range strings.Split(issue.Description, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				// Preserve blank lines in content
				otherLines = append(otherLines, line)
				continue
			}

			// Check if this is an attachment field line
			colonIdx := strings.Index(trimmed, ":")
			if colonIdx == -1 {
				otherLines = append(otherLines, line)
				continue
			}

			key := strings.ToLower(strings.TrimSpace(trimmed[:colonIdx]))
			if !attachmentKeys[key] {
				otherLines = append(otherLines, line)
			}
			// Skip attachment field lines - they'll be replaced
		}
	}

	// Build new description: attachment fields first, then other content
	formatted := FormatAttachmentFields(fields)

	// Trim trailing blank lines from other content
	for len(otherLines) > 0 && strings.TrimSpace(otherLines[len(otherLines)-1]) == "" {
		otherLines = otherLines[:len(otherLines)-1]
	}
	// Trim leading blank lines from other content
	for len(otherLines) > 0 && strings.TrimSpace(otherLines[0]) == "" {
		otherLines = otherLines[1:]
	}

	if formatted == "" {
		return strings.Join(otherLines, "\n")
	}
	if len(otherLines) == 0 {
		return formatted
	}

	return formatted + "\n\n" + strings.Join(otherLines, "\n")
}

// MRFields holds the structured fields for a merge-request issue.
// These fields are stored as key: value lines in the issue description.
type MRFields struct {
	Branch      string // Source branch name (e.g., "polecat/Nux/gt-xyz")
	Target      string // Target branch (e.g., "main" or "integration/gt-epic")
	SourceIssue string // The work item being merged (e.g., "gt-xyz")
	Worker      string // Who did the work
	Rig         string // Which rig
	MergeCommit string // SHA of merge commit (set on close)
	CloseReason string // Reason for closing: merged, rejected, conflict, superseded
	AgentBead   string // Agent bead ID that created this MR (for traceability)
	ConvoyID    string // Parent convoy ID if part of a convoy (for priority scoring)
	RetryCount  int    // Conflict retry count for priority penalty
}

// ParseMRFields extracts structured merge-request fields from an issue's description.
// Fields are expected as "key: value" lines, with optional prose text mixed in.
// Returns nil if no MR fields are found.
func ParseMRFields(issue *Issue) *MRFields {
	if issue == nil || issue.Description == "" {
		return nil
	}

	fields := &MRFields{}
	hasFields := false

	for _, line := range strings.Split(issue.Description, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for "key: value" pattern
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		if value == "" {
			continue
		}

		// Map keys to fields (case-insensitive)
		switch strings.ToLower(key) {
		case "branch":
			fields.Branch = value
			hasFields = true
		case "target":
			fields.Target = value
			hasFields = true
		case "source_issue", "source-issue", "sourceissue":
			fields.SourceIssue = value
			hasFields = true
		case "worker":
			fields.Worker = value
			hasFields = true
		case "rig":
			fields.Rig = value
			hasFields = true
		case "merge_commit", "merge-commit", "mergecommit":
			fields.MergeCommit = value
			hasFields = true
		case "close_reason", "close-reason", "closereason":
			fields.CloseReason = value
			hasFields = true
		case "agent_bead", "agent-bead", "agentbead":
			fields.AgentBead = value
			hasFields = true
		case "convoy", "convoy_id", "convoy-id", "convoyid":
			fields.ConvoyID = value
			hasFields = true
		case "retry_count", "retry-count", "retrycount":
			// Parse integer, default to 0 on error
			var count int
			fmt.Sscanf(value, "%d", &count)
			fields.RetryCount = count
			hasFields = true
		}
	}

	if !hasFields {
		return nil
	}
	return fields
}

// FormatMRFields formats MRFields as a string suitable for an issue description.
// Only non-empty fields are included.
func FormatMRFields(fields *MRFields) string {
	if fields == nil {
		return ""
	}

	var lines []string

	if fields.Branch != "" {
		lines = append(lines, "branch: "+fields.Branch)
	}
	if fields.Target != "" {
		lines = append(lines, "target: "+fields.Target)
	}
	if fields.SourceIssue != "" {
		lines = append(lines, "source_issue: "+fields.SourceIssue)
	}
	if fields.Worker != "" {
		lines = append(lines, "worker: "+fields.Worker)
	}
	if fields.Rig != "" {
		lines = append(lines, "rig: "+fields.Rig)
	}
	if fields.MergeCommit != "" {
		lines = append(lines, "merge_commit: "+fields.MergeCommit)
	}
	if fields.CloseReason != "" {
		lines = append(lines, "close_reason: "+fields.CloseReason)
	}
	if fields.AgentBead != "" {
		lines = append(lines, "agent_bead: "+fields.AgentBead)
	}
	if fields.ConvoyID != "" {
		lines = append(lines, "convoy_id: "+fields.ConvoyID)
	}
	if fields.RetryCount > 0 {
		lines = append(lines, fmt.Sprintf("retry_count: %d", fields.RetryCount))
	}

	return strings.Join(lines, "\n")
}

// SetMRFields updates an issue's description with the given MR fields.
// Existing MR field lines are replaced; other content is preserved.
// Returns the new description string.
func SetMRFields(issue *Issue, fields *MRFields) string {
	if issue == nil {
		return FormatMRFields(fields)
	}

	// Known MR field keys (lowercase)
	mrKeys := map[string]bool{
		"branch":       true,
		"target":       true,
		"source_issue": true,
		"source-issue": true,
		"sourceissue":  true,
		"worker":       true,
		"rig":          true,
		"merge_commit": true,
		"merge-commit": true,
		"mergecommit":  true,
		"close_reason": true,
		"close-reason": true,
		"closereason":  true,
		"agent_bead":   true,
		"agent-bead":   true,
		"agentbead":    true,
		"convoy":       true,
		"convoy_id":    true,
		"convoy-id":    true,
		"convoyid":     true,
		"retry_count":  true,
		"retry-count":  true,
		"retrycount":   true,
	}

	// Collect non-MR lines from existing description
	var otherLines []string
	if issue.Description != "" {
		for _, line := range strings.Split(issue.Description, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				// Preserve blank lines in content
				otherLines = append(otherLines, line)
				continue
			}

			// Check if this is an MR field line
			colonIdx := strings.Index(trimmed, ":")
			if colonIdx == -1 {
				otherLines = append(otherLines, line)
				continue
			}

			key := strings.ToLower(strings.TrimSpace(trimmed[:colonIdx]))
			if !mrKeys[key] {
				otherLines = append(otherLines, line)
			}
			// Skip MR field lines - they'll be replaced
		}
	}

	// Build new description: MR fields first, then other content
	formatted := FormatMRFields(fields)

	// Trim trailing blank lines from other content
	for len(otherLines) > 0 && strings.TrimSpace(otherLines[len(otherLines)-1]) == "" {
		otherLines = otherLines[:len(otherLines)-1]
	}
	// Trim leading blank lines from other content
	for len(otherLines) > 0 && strings.TrimSpace(otherLines[0]) == "" {
		otherLines = otherLines[1:]
	}

	if formatted == "" {
		return strings.Join(otherLines, "\n")
	}
	if len(otherLines) == 0 {
		return formatted
	}

	return formatted + "\n\n" + strings.Join(otherLines, "\n")
}

// SynthesisFields holds structured fields for synthesis beads.
// These fields track the synthesis step in a convoy workflow.
type SynthesisFields struct {
	ConvoyID   string `json:"convoy_id"`   // Parent convoy ID
	ReviewID   string `json:"review_id"`   // Review ID for output paths
	OutputPath string `json:"output_path"` // Path to synthesis output file
	Formula    string `json:"formula"`     // Formula name (if from formula)
}

// ParseSynthesisFields extracts synthesis fields from an issue's description.
// Fields are expected as "key: value" lines. Returns nil if no fields found.
func ParseSynthesisFields(issue *Issue) *SynthesisFields {
	if issue == nil || issue.Description == "" {
		return nil
	}

	fields := &SynthesisFields{}
	hasFields := false

	for _, line := range strings.Split(issue.Description, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		if value == "" {
			continue
		}

		switch strings.ToLower(key) {
		case "convoy", "convoy_id", "convoy-id":
			fields.ConvoyID = value
			hasFields = true
		case "review_id", "review-id", "reviewid":
			fields.ReviewID = value
			hasFields = true
		case "output_path", "output-path", "outputpath":
			fields.OutputPath = value
			hasFields = true
		case "formula":
			fields.Formula = value
			hasFields = true
		}
	}

	if !hasFields {
		return nil
	}
	return fields
}

// FormatSynthesisFields formats SynthesisFields as a string for issue description.
func FormatSynthesisFields(fields *SynthesisFields) string {
	if fields == nil {
		return ""
	}

	var lines []string
	if fields.ConvoyID != "" {
		lines = append(lines, "convoy: "+fields.ConvoyID)
	}
	if fields.ReviewID != "" {
		lines = append(lines, "review_id: "+fields.ReviewID)
	}
	if fields.OutputPath != "" {
		lines = append(lines, "output_path: "+fields.OutputPath)
	}
	if fields.Formula != "" {
		lines = append(lines, "formula: "+fields.Formula)
	}

	return strings.Join(lines, "\n")
}

// RoleConfig holds structured lifecycle configuration for role beads.
// These fields are stored as "key: value" lines in the role bead description.
// This enables agents to self-register their lifecycle configuration,
// replacing hardcoded identity string parsing in the daemon.
type RoleConfig struct {
	// SessionPattern defines how to derive tmux session name.
	// Supports placeholders: {rig}, {name}, {role}
	// Examples: "gt-mayor", "gt-{rig}-{role}", "gt-{rig}-{name}"
	SessionPattern string

	// WorkDirPattern defines the working directory relative to town root.
	// Supports placeholders: {town}, {rig}, {name}, {role}
	// Examples: "{town}", "{town}/{rig}", "{town}/{rig}/polecats/{name}"
	WorkDirPattern string

	// NeedsPreSync indicates whether workspace needs git sync before starting.
	// True for agents with persistent clones (refinery, crew, polecat).
	NeedsPreSync bool

	// StartCommand is the command to run after creating the session.
	// Default: "exec claude --dangerously-skip-permissions"
	StartCommand string

	// EnvVars are additional environment variables to set in the session.
	// Stored as "key=value" pairs.
	EnvVars map[string]string
}

// ParseRoleConfig extracts RoleConfig from a role bead's description.
// Fields are expected as "key: value" lines. Returns nil if no config found.
func ParseRoleConfig(description string) *RoleConfig {
	config := &RoleConfig{
		EnvVars: make(map[string]string),
	}
	hasFields := false

	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		if value == "" || value == "null" {
			continue
		}

		switch strings.ToLower(key) {
		case "session_pattern", "session-pattern", "sessionpattern":
			config.SessionPattern = value
			hasFields = true
		case "work_dir_pattern", "work-dir-pattern", "workdirpattern", "workdir_pattern":
			config.WorkDirPattern = value
			hasFields = true
		case "needs_pre_sync", "needs-pre-sync", "needspresync":
			config.NeedsPreSync = strings.ToLower(value) == "true"
			hasFields = true
		case "start_command", "start-command", "startcommand":
			config.StartCommand = value
			hasFields = true
		case "env_var", "env-var", "envvar":
			// Format: "env_var: KEY=VALUE"
			if eqIdx := strings.Index(value, "="); eqIdx != -1 {
				envKey := strings.TrimSpace(value[:eqIdx])
				envVal := strings.TrimSpace(value[eqIdx+1:])
				config.EnvVars[envKey] = envVal
				hasFields = true
			}
		}
	}

	if !hasFields {
		return nil
	}
	return config
}

// FormatRoleConfig formats RoleConfig as a string suitable for a role bead description.
// Only non-empty/non-default fields are included.
func FormatRoleConfig(config *RoleConfig) string {
	if config == nil {
		return ""
	}

	var lines []string

	if config.SessionPattern != "" {
		lines = append(lines, "session_pattern: "+config.SessionPattern)
	}
	if config.WorkDirPattern != "" {
		lines = append(lines, "work_dir_pattern: "+config.WorkDirPattern)
	}
	if config.NeedsPreSync {
		lines = append(lines, "needs_pre_sync: true")
	}
	if config.StartCommand != "" {
		lines = append(lines, "start_command: "+config.StartCommand)
	}
	for k, v := range config.EnvVars {
		lines = append(lines, "env_var: "+k+"="+v)
	}

	return strings.Join(lines, "\n")
}

// ExpandRolePattern expands placeholders in a pattern string.
// Supported placeholders: {town}, {rig}, {name}, {role}
func ExpandRolePattern(pattern, townRoot, rig, name, role string) string {
	result := pattern
	result = strings.ReplaceAll(result, "{town}", townRoot)
	result = strings.ReplaceAll(result, "{rig}", rig)
	result = strings.ReplaceAll(result, "{name}", name)
	result = strings.ReplaceAll(result, "{role}", role)
	return result
}

// MRScoreConfig contains tunable weights for MR priority scoring.
// All weights are designed so higher scores = higher priority (process first).
type MRScoreConfig struct {
	BaseScore       float64 // Starting score (default: 1000)
	ConvoyAgeWeight float64 // Points per hour of convoy age (default: 10)
	PriorityWeight  float64 // Multiplied by (4 - priority) (default: 100)
	RetryPenalty    float64 // Points subtracted per retry (default: 50)
	MRAgeWeight     float64 // Points per hour since MR creation (default: 1)
	MaxRetryPenalty float64 // Cap on retry penalty (default: 300)
}

// DefaultMRScoreConfig returns sensible defaults for MR scoring.
func DefaultMRScoreConfig() MRScoreConfig {
	return MRScoreConfig{
		BaseScore:       1000.0,
		ConvoyAgeWeight: 10.0,
		PriorityWeight:  100.0,
		RetryPenalty:    50.0,
		MRAgeWeight:     1.0,
		MaxRetryPenalty: 300.0,
	}
}

// MRScoreInput contains the data needed to score an MR issue.
type MRScoreInput struct {
	Priority        int        // Issue priority (0=P0/critical, 4=P4/backlog)
	CreatedAt       string     // ISO 8601 timestamp of MR creation
	ConvoyCreatedAt string     // ISO 8601 timestamp of convoy creation (empty if standalone)
	RetryCount      int        // Conflict retry count
}

// ScoreMRIssue calculates the priority score for a merge-request issue.
// Higher scores mean higher priority (process first).
//
// The scoring formula:
//
//	score = BaseScore
//	      + ConvoyAgeWeight * hoursOld(convoy)       // Prevent convoy starvation
//	      + PriorityWeight * (4 - priority)          // P0=+400, P4=+0
//	      - min(RetryPenalty * retryCount, MaxRetryPenalty)  // Prevent thrashing
//	      + MRAgeWeight * hoursOld(MR)               // FIFO tiebreaker
func ScoreMRIssue(input MRScoreInput, config MRScoreConfig) float64 {
	now := time.Now()

	score := config.BaseScore

	// Convoy age factor: prevent starvation of old convoys
	if input.ConvoyCreatedAt != "" {
		if convoyTime, err := parseISOTime(input.ConvoyCreatedAt); err == nil {
			convoyHours := now.Sub(convoyTime).Hours()
			if convoyHours > 0 {
				score += config.ConvoyAgeWeight * convoyHours
			}
		}
	}

	// Priority factor: P0 (0) gets +400, P4 (4) gets +0
	priorityBonus := 4 - input.Priority
	if priorityBonus < 0 {
		priorityBonus = 0 // Clamp for invalid priorities > 4
	}
	if priorityBonus > 4 {
		priorityBonus = 4 // Clamp for invalid priorities < 0
	}
	score += config.PriorityWeight * float64(priorityBonus)

	// Retry penalty: prevent thrashing on repeatedly failing MRs
	retryPenalty := config.RetryPenalty * float64(input.RetryCount)
	if retryPenalty > config.MaxRetryPenalty {
		retryPenalty = config.MaxRetryPenalty
	}
	score -= retryPenalty

	// MR age factor: FIFO ordering as tiebreaker
	if input.CreatedAt != "" {
		if mrTime, err := parseISOTime(input.CreatedAt); err == nil {
			mrHours := now.Sub(mrTime).Hours()
			if mrHours > 0 {
				score += config.MRAgeWeight * mrHours
			}
		}
	}

	return score
}

// ScoreMRIssueWithDefaults is a convenience wrapper using default config.
func ScoreMRIssueWithDefaults(input MRScoreInput) float64 {
	return ScoreMRIssue(input, DefaultMRScoreConfig())
}

// parseISOTime parses an ISO 8601 timestamp string.
func parseISOTime(s string) (time.Time, error) {
	// Try RFC3339 first (most common)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}
	// Try date-only
	return time.Parse("2006-01-02", s)
}
