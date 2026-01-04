# Agent Monitoring System

The monitoring package provides real-time status inference for agents in Gas Town by analyzing their activity patterns.

## Overview

Agent status can come from three sources:
- **boss** - Explicitly set by Witness/Mayor
- **self** - Agent reports its own status
- **inferred** - Detected from pane output patterns or idle detection

## Status Types

```
available  - Agent is available for work
working    - Agent is actively working
thinking   - Agent is thinking/planning
blocked    - Agent is blocked waiting
idle       - Agent has been inactive
error      - Agent encountered an error
offline    - Agent tmux session not running
```

## Usage

### Basic Pattern Detection

```go
detector := monitoring.NewPatternDetector()
status, pattern := detector.DetectStatus("agent-1", "Thinking...")
// status = "thinking", pattern = "Thinking"
```

### Tracking Agent Status

```go
tracker := monitoring.NewTracker(tmux)

// Infer status from pane output
agentStatus := tracker.InferStatus(address, tmuxSession, agentState, hasWork)

// Set explicit status
tracker.SetStatus(address, monitoring.StatusWorking, monitoring.SourceBoss)

// Retrieve status
status := tracker.GetStatus(address)
```

### Integration with `gt status`

Use the `--monitor` flag to infer agent activity status:

```bash
gt status --monitor
```

The monitor status will appear in the agent details:

```
gt-gastown-witness running
  hook: gt-5abc → Fix compilation error
  activity: thinking (Thinking)
  mail: 📬 2 unread
```

## Pattern Detection

The detector uses regex patterns to identify status from pane output:

| Pattern | Status | Triggers |
|---------|--------|----------|
| `Thinking...` | thinking | Agent is thinking/planning |
| `BLOCKED:` | blocked | Agent is explicitly blocked |
| `Error:` | error | Error encountered |
| `Waiting for` | blocked | Waiting for resource |
| `Running\|processing` | working | Active execution |
| `Compiling\|building` | working | Build in progress |
| `Failed\|failure` | error | Failure occurred |
| `Panic\|fatal` | error | Fatal error |

## Idle Detection

Agents with no pane output for 60+ seconds (configurable via `IdleThreshold`) are marked as idle.

## Implementation Details

### PatternDetector

Tracks activity history and detects status patterns:
- `DetectStatus(address, paneOutput)` - Detect status from output
- `GetIdleStatus(address, now)` - Check if agent is idle
- `RecordActivity(address)` - Mark activity timestamp
- `LastActivity(address)` - Retrieve activity timestamp

### Tracker

Thread-safe tracker for multiple agents:
- `InferStatus(address, session, agentState, hasWork)` - Infer from pane + state
- `SetStatus(address, status, source)` - Explicitly set status
- `GetStatus(address)` - Get current status
- `GetAllStatuses()` - Get all tracked statuses
- `Clear()` - Reset all tracking

## Testing

```bash
go test ./internal/monitoring/... -v
```

All tests verify pattern detection, idle detection, activity tracking, and multi-agent scenarios.

## Integration Points

- **status.go** - Uses tracker to infer agent activity in `gt status --monitor`
- **AgentRuntime struct** - Extended with `MonitorStatus` and `ActivityStatus` fields
- **discoverGlobalAgents()** - Calls tracker for each global agent
- **discoverRigAgents()** - Calls tracker for each rig agent
- **renderAgentDetails()** - Displays monitor status in output

## Future Enhancements

- Persistent activity logging
- Configurable pattern registry
- Agent performance metrics
- Status change notifications
- Custom idle thresholds per agent type
