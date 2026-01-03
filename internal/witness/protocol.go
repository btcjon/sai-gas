// Package witness provides the polecat monitoring agent.
package witness

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Protocol message patterns for Witness inbox routing.
var (
	// POLECAT_DONE <name> - polecat signaling work completion
	PatternPolecatDone = regexp.MustCompile(`^POLECAT_DONE\s+(\S+)`)

	// LIFECYCLE:Shutdown <name> - daemon-triggered polecat shutdown
	PatternLifecycleShutdown = regexp.MustCompile(`^LIFECYCLE:Shutdown\s+(\S+)`)

	// HELP: <topic> - polecat requesting intervention
	PatternHelp = regexp.MustCompile(`^HELP:\s+(.+)`)

	// MERGED <name> - refinery confirms branch merged
	PatternMerged = regexp.MustCompile(`^MERGED\s+(\S+)`)

	// CONFLICT_DISPATCH <mr-id> - refinery requesting conflict resolution dispatch
	// Sent when an MR has merge conflicts that need a polecat to resolve.
	PatternConflictDispatch = regexp.MustCompile(`^CONFLICT_DISPATCH\s+(\S+)`)

	// HANDOFF - session continuity message
	PatternHandoff = regexp.MustCompile(`^🤝\s*HANDOFF`)

	// SWARM_START - mayor initiating batch work
	PatternSwarmStart = regexp.MustCompile(`^SWARM_START`)
)

// ProtocolType identifies the type of protocol message.
type ProtocolType string

const (
	ProtoPolecatDone       ProtocolType = "polecat_done"
	ProtoLifecycleShutdown ProtocolType = "lifecycle_shutdown"
	ProtoHelp              ProtocolType = "help"
	ProtoMerged            ProtocolType = "merged"
	ProtoConflictDispatch  ProtocolType = "conflict_dispatch"
	ProtoHandoff           ProtocolType = "handoff"
	ProtoSwarmStart        ProtocolType = "swarm_start"
	ProtoUnknown           ProtocolType = "unknown"
)

// PolecatDonePayload contains parsed data from a POLECAT_DONE message.
type PolecatDonePayload struct {
	PolecatName string
	Exit        string // MERGED, ESCALATED, DEFERRED
	IssueID     string
	MRID        string
	Branch      string
}

// HelpPayload contains parsed data from a HELP message.
type HelpPayload struct {
	Topic       string
	Agent       string
	IssueID     string
	Problem     string
	Tried       string
	RequestedAt time.Time
}

// MergedPayload contains parsed data from a MERGED message.
type MergedPayload struct {
	PolecatName string
	Branch      string
	IssueID     string
	MergedAt    time.Time
}

// SwarmStartPayload contains parsed data from a SWARM_START message.
type SwarmStartPayload struct {
	SwarmID   string
	BeadIDs   []string
	Total     int
	StartedAt time.Time
}

// ConflictDispatchPayload contains parsed data from a CONFLICT_DISPATCH message.
// This is sent by the Refinery when an MR has merge conflicts that need resolution.
type ConflictDispatchPayload struct {
	MRID            string    // The MR ID that has conflicts
	Branch          string    // Source branch that needs rebasing
	TargetBranch    string    // Target branch (usually "main")
	ConflictSHA     string    // SHA of target when conflict detected
	SourceIssue     string    // Original work issue being merged
	Worker          string    // Original polecat that did the work
	RetryCount      int       // How many times this has been retried
	Rig             string    // Which rig this is in
	RequestedAt     time.Time // When conflict was detected
}

// ClassifyMessage determines the protocol type from a message subject.
func ClassifyMessage(subject string) ProtocolType {
	switch {
	case PatternPolecatDone.MatchString(subject):
		return ProtoPolecatDone
	case PatternLifecycleShutdown.MatchString(subject):
		return ProtoLifecycleShutdown
	case PatternHelp.MatchString(subject):
		return ProtoHelp
	case PatternMerged.MatchString(subject):
		return ProtoMerged
	case PatternConflictDispatch.MatchString(subject):
		return ProtoConflictDispatch
	case PatternHandoff.MatchString(subject):
		return ProtoHandoff
	case PatternSwarmStart.MatchString(subject):
		return ProtoSwarmStart
	default:
		return ProtoUnknown
	}
}

// ParsePolecatDone extracts payload from a POLECAT_DONE message.
// Subject format: POLECAT_DONE <polecat-name>
// Body format:
//
//	Exit: MERGED|ESCALATED|DEFERRED
//	Issue: <issue-id>
//	MR: <mr-id>
//	Branch: <branch>
func ParsePolecatDone(subject, body string) (*PolecatDonePayload, error) {
	matches := PatternPolecatDone.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid POLECAT_DONE subject: %s", subject)
	}

	payload := &PolecatDonePayload{
		PolecatName: matches[1],
	}

	// Parse body for structured fields
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Exit:") {
			payload.Exit = strings.TrimSpace(strings.TrimPrefix(line, "Exit:"))
		} else if strings.HasPrefix(line, "Issue:") {
			payload.IssueID = strings.TrimSpace(strings.TrimPrefix(line, "Issue:"))
		} else if strings.HasPrefix(line, "MR:") {
			payload.MRID = strings.TrimSpace(strings.TrimPrefix(line, "MR:"))
		} else if strings.HasPrefix(line, "Branch:") {
			payload.Branch = strings.TrimSpace(strings.TrimPrefix(line, "Branch:"))
		}
	}

	return payload, nil
}

// ParseHelp extracts payload from a HELP message.
// Subject format: HELP: <topic>
// Body format:
//
//	Agent: <agent-id>
//	Issue: <issue-id>
//	Problem: <description>
//	Tried: <what was attempted>
func ParseHelp(subject, body string) (*HelpPayload, error) {
	matches := PatternHelp.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid HELP subject: %s", subject)
	}

	payload := &HelpPayload{
		Topic:       matches[1],
		RequestedAt: time.Now(),
	}

	// Parse body for structured fields
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Agent:") {
			payload.Agent = strings.TrimSpace(strings.TrimPrefix(line, "Agent:"))
		} else if strings.HasPrefix(line, "Issue:") {
			payload.IssueID = strings.TrimSpace(strings.TrimPrefix(line, "Issue:"))
		} else if strings.HasPrefix(line, "Problem:") {
			payload.Problem = strings.TrimSpace(strings.TrimPrefix(line, "Problem:"))
		} else if strings.HasPrefix(line, "Tried:") {
			payload.Tried = strings.TrimSpace(strings.TrimPrefix(line, "Tried:"))
		}
	}

	return payload, nil
}

// ParseMerged extracts payload from a MERGED message.
// Subject format: MERGED <polecat-name>
// Body format:
//
//	Branch: <branch>
//	Issue: <issue-id>
//	Merged-At: <timestamp>
func ParseMerged(subject, body string) (*MergedPayload, error) {
	matches := PatternMerged.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid MERGED subject: %s", subject)
	}

	payload := &MergedPayload{
		PolecatName: matches[1],
	}

	// Parse body for structured fields
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Branch:") {
			payload.Branch = strings.TrimSpace(strings.TrimPrefix(line, "Branch:"))
		} else if strings.HasPrefix(line, "Issue:") {
			payload.IssueID = strings.TrimSpace(strings.TrimPrefix(line, "Issue:"))
		} else if strings.HasPrefix(line, "Merged-At:") {
			ts := strings.TrimSpace(strings.TrimPrefix(line, "Merged-At:"))
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				payload.MergedAt = t
			}
		}
	}

	return payload, nil
}

// ParseSwarmStart extracts payload from a SWARM_START message.
// Body format is JSON: {"swarm_id": "batch-123", "beads": ["bd-a", "bd-b"]}
func ParseSwarmStart(body string) (*SwarmStartPayload, error) {
	payload := &SwarmStartPayload{
		StartedAt: time.Now(),
	}

	// Parse the JSON-like body (simplified parsing for key-value extraction)
	// Full JSON parsing would require encoding/json import
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SwarmID:") || strings.HasPrefix(line, "swarm_id:") {
			payload.SwarmID = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "SwarmID:"), "swarm_id:"))
		} else if strings.HasPrefix(line, "Total:") {
			fmt.Sscanf(line, "Total: %d", &payload.Total)
		}
	}

	return payload, nil
}

// ParseConflictDispatch extracts payload from a CONFLICT_DISPATCH message.
// Subject format: CONFLICT_DISPATCH <mr-id>
// Body format:
//
//	Branch: <source-branch>
//	Target: <target-branch>
//	ConflictSHA: <sha>
//	SourceIssue: <issue-id>
//	Worker: <polecat-name>
//	Rig: <rig-name>
//	RetryCount: <count>
func ParseConflictDispatch(subject, body string) (*ConflictDispatchPayload, error) {
	matches := PatternConflictDispatch.FindStringSubmatch(subject)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid CONFLICT_DISPATCH subject: %s", subject)
	}

	payload := &ConflictDispatchPayload{
		MRID:        matches[1],
		RequestedAt: time.Now(),
	}

	// Parse body for structured fields
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Branch:") {
			payload.Branch = strings.TrimSpace(strings.TrimPrefix(line, "Branch:"))
		} else if strings.HasPrefix(line, "Target:") {
			payload.TargetBranch = strings.TrimSpace(strings.TrimPrefix(line, "Target:"))
		} else if strings.HasPrefix(line, "ConflictSHA:") {
			payload.ConflictSHA = strings.TrimSpace(strings.TrimPrefix(line, "ConflictSHA:"))
		} else if strings.HasPrefix(line, "SourceIssue:") {
			payload.SourceIssue = strings.TrimSpace(strings.TrimPrefix(line, "SourceIssue:"))
		} else if strings.HasPrefix(line, "Worker:") {
			payload.Worker = strings.TrimSpace(strings.TrimPrefix(line, "Worker:"))
		} else if strings.HasPrefix(line, "Rig:") {
			payload.Rig = strings.TrimSpace(strings.TrimPrefix(line, "Rig:"))
		} else if strings.HasPrefix(line, "RetryCount:") {
			fmt.Sscanf(strings.TrimPrefix(line, "RetryCount:"), "%d", &payload.RetryCount)
		}
	}

	return payload, nil
}

// CleanupWispLabels generates labels for a cleanup wisp.
func CleanupWispLabels(polecatName, state string) []string {
	return []string{
		"cleanup",
		fmt.Sprintf("polecat:%s", polecatName),
		fmt.Sprintf("state:%s", state),
	}
}

// SwarmWispLabels generates labels for a swarm tracking wisp.
func SwarmWispLabels(swarmID string, total, completed int, startTime time.Time) []string {
	return []string{
		"swarm",
		fmt.Sprintf("swarm_id:%s", swarmID),
		fmt.Sprintf("total:%d", total),
		fmt.Sprintf("completed:%d", completed),
		fmt.Sprintf("start:%s", startTime.Format(time.RFC3339)),
	}
}

// HelpAssessment represents the Witness's assessment of a help request.
type HelpAssessment struct {
	CanHelp     bool
	HelpAction  string // What the Witness can do to help
	NeedsEscalation bool
	EscalationReason string
}

// AssessHelpRequest provides guidance for the Witness to assess a help request.
// This is a template/guide - actual assessment is done by the Claude agent.
func AssessHelpRequest(payload *HelpPayload) *HelpAssessment {
	assessment := &HelpAssessment{}

	// Heuristics for common help requests that Witness can handle
	topic := strings.ToLower(payload.Topic)
	problem := strings.ToLower(payload.Problem)

	// Git issues - Witness can often help
	if strings.Contains(topic, "git") || strings.Contains(problem, "git") {
		if strings.Contains(problem, "conflict") {
			assessment.CanHelp = false
			assessment.NeedsEscalation = true
			assessment.EscalationReason = "Git conflicts require human review"
		} else if strings.Contains(problem, "push") || strings.Contains(problem, "fetch") {
			assessment.CanHelp = true
			assessment.HelpAction = "Check git remote status and network connectivity"
		}
	}

	// Test failures - usually need escalation
	if strings.Contains(topic, "test") || strings.Contains(problem, "test fail") {
		assessment.CanHelp = false
		assessment.NeedsEscalation = true
		assessment.EscalationReason = "Test failures require investigation"
	}

	// Build issues - Witness can check basics
	if strings.Contains(topic, "build") || strings.Contains(problem, "compile") {
		assessment.CanHelp = true
		assessment.HelpAction = "Verify dependencies and build configuration"
	}

	// Requirements unclear - always escalate
	if strings.Contains(topic, "unclear") || strings.Contains(problem, "requirement") ||
		strings.Contains(problem, "don't understand") {
		assessment.CanHelp = false
		assessment.NeedsEscalation = true
		assessment.EscalationReason = "Requirements clarification needed from Mayor"
	}

	// Default: escalate if we don't recognize the pattern
	if !assessment.CanHelp && !assessment.NeedsEscalation {
		assessment.NeedsEscalation = true
		assessment.EscalationReason = "Unknown help request type"
	}

	return assessment
}
