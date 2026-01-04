# Ephemeral Polecat Workflow

## Overview

Polecats in Gas Town are **ephemeral agents**: they are spawned to work on a single issue, submit their merge request, then are recyclable (ready for cleanup). This is the "rebase-as-work" architecture - conflicts are treated as new work assignments, not as blockers.

## Key Concept: Rebase-as-Work

When a polecat's merge request encounters conflicts:

1. **Refinery detects the conflict** during merge attempt
2. **New conflict-resolution task is created** in beads with metadata
3. **Refinery automatically creates a high-priority MR** for the task
4. **Conflict resolution becomes another work assignment** (same polecat or another)
5. **Original polecat is clean and recyclable** (no long-lived state)

This decouples conflict resolution from the original work, preventing polecats from becoming permanently bound to issues.

## Polecat Lifecycle States

A polecat's state machine tracks its position in the workflow:

```
spawning
    ↓
working     ← Primary state: actively working on the assigned issue
    ↓
mr_submitted ← Transitional: MR created and pushed to origin
    ↓
recyclable  ← Exit state: ready for cleanup by Witness
    ↓
recycled    ← (Witness-managed: cleaned up, removed)
```

### State Descriptions

| State | Meaning | Duration | Exit Condition |
|-------|---------|----------|----------------|
| `working` | Actively working on the issue | Minutes to hours | `gt done --exit COMPLETED` creates MR |
| `mr_submitted` | MR queued, awaiting refinery confirmation | Seconds to minutes | Branch confirmed pushed to origin |
| `recyclable` | Ready for cleanup | Minimal | Witness removes the polecat/worktree |
| `stuck` | Blocked and needs help | Indefinite | Manual intervention or Witness escalation |

## MR Submission as Exit Point

When a polecat finishes work:

```bash
# 1. Complete the work
bd close <final-step> --continue

# 2. Submit and exit
gt done --exit COMPLETED

# This:
# - Commits all changes
# - Creates the MR in the queue
# - Syncs beads
# - Sets polecat state to mr_submitted
# - Signals Witness that work is done
```

The polecat **does not wait for the merge to complete**. This is ephemeral behavior:
- Work is "done" when the branch is pushed and queued
- The refinery handles merging
- If conflicts arise, that becomes a separate conflict-resolution task

## Conflict Resolution Flow

### Detection Phase

1. **Refinery attempts to merge** the polecat's branch
2. **Git merge fails** with conflicts
3. **Refinery detects** the conflict (exit code or marker files)
4. **MR status updated** to `closed` with reason `conflict`

### Resolution Task Creation

Refinery creates a conflict-resolution task:

```
Title: "Resolve merge conflicts: <original-issue>"
Description:
  ## Metadata
  - Original MR: mr-1234-abcd
  - Branch: polecat/Toast-abc123
  - Conflict with: main@abc12345def
  - Original issue: gt-xyz
  - Retry count: 1
  - ...
```

The task metadata captures:
- **OriginalMRID**: Link back to the failed MR
- **Branch**: The source branch that needs rebasing
- **TargetBranch@SHA**: What it conflicted with
- **SourceIssue**: The original work that triggered this
- **RetryCount**: How many times this MR has been retried

### Assignment Phase

The conflict-resolution task is:
1. **Created as a bead** in the rig's beads system
2. **Assigned to a polecat** (via sling/queue)
3. **Given high priority** (retry count boosts the priority score)
4. **Isolated from the original polecat** (separate work item)

The polecat resolving conflicts:
1. Checks out the conflicted branch
2. Rebases onto the current target branch
3. Resolves merge conflicts (automatic git tools + manual edits)
4. Pushes the rebased branch
5. Submits a new MR

### Eventual Success

Once the conflict is resolved and the rebase MR merges:
1. **Original work is on main** (via the rebased branch)
2. **Conflict-resolution task closes** (merged)
3. **All polecats involved are recyclable**

## Priority Scoring

The merge queue uses a deterministic priority function to order MRs. Higher scores are processed first.

### Scoring Formula

```
score = 1000                                           (BaseScore)
      + 10 * hoursOld(convoy)                         (ConvoyAgeWeight)
      + 100 * (4 - priority)                          (PriorityWeight: P0→+400, P4→+0)
      - min(50 * retryCount, 300)                     (RetryPenalty: capped at 300)
      + 1 * hoursOld(MR)                              (MRAgeWeight: FIFO tiebreaker)
```

### Design Principles

1. **Deterministic**: Same inputs always produce the same score (no randomness, uses explicit time)

2. **Convoy Starvation Prevention**: Older convoys accumulate points and eventually beat fresh high-priority items
   - 24h convoy: +240 points
   - 48h convoy: +480 points
   - A 48h P4 convoy (1480 points) beats a fresh P0 (1400 points)

3. **Priority Respect**: Within similar convoy ages, P0 bugs beat P4 backlog
   - Fresh P0: 1400 points
   - Fresh P4: 1000 points
   - Difference: 400 points

4. **Thrashing Prevention**: MRs that repeatedly fail get deprioritized
   - First attempt: 0 penalty
   - 3 retries: -150 penalty
   - 6+ retries: -300 (capped)
   - This gives the repository time to stabilize before retrying

5. **FIFO Fairness**: Within the same convoy/priority/retry state, older MRs go first
   - MR age adds 1 point per hour
   - Minor factor, breaks ties

### Example Scores

| Scenario | Score | Notes |
|----------|-------|-------|
| Fresh P0, standalone | 1400 | 1000 + (4-0)*100 |
| Fresh P4, standalone | 1000 | 1000 + (4-4)*100 |
| Fresh P2, 24h convoy | 1440 | 1000 + (4-2)*100 + 10*24 |
| Fresh P4, 48h convoy | 1480 | 1000 + (4-4)*100 + 10*48 |
| P0, 1st retry, 24h convoy | 1550 | 1400 + 240 - 50 |
| P0, 6+ retries, 24h convoy | 1640 | 1400 + 240 - 300 |

### Tuning

All weights are configurable via `ScoreConfig`:

| Weight | Default | Meaning |
|--------|---------|---------|
| `BaseScore` | 1000.0 | Keep all scores positive |
| `ConvoyAgeWeight` | 10.0 | 10 points per hour of convoy age |
| `PriorityWeight` | 100.0 | P0 gets 400 pts, P4 gets 0 pts |
| `RetryPenalty` | 50.0 | Lose 50 pts per retry |
| `MRAgeWeight` | 1.0 | 1 point per hour of MR age |
| `MaxRetryPenalty` | 300.0 | Cap total retry penalty at 300 |

These defaults ensure:
- A 48-hour convoy beats any standalone priority (convoy starvation prevention)
- Priority differences dominate within the same convoy age
- Retry penalties are significant but bounded (eventual progress guaranteed)

## Example Timeline

```
T=0:00   Polecat spawned, receives issue gt-abc
T=0:05   Starts work (state: working)
T=1:30   Completes implementation
T=1:31   Runs gt done → MR submitted, state: mr_submitted
T=1:33   Branch confirmed at origin, state: recyclable
T=1:35   Refinery picks up MR for processing

T=1:40   Refinery attempts merge → CONFLICT
         Refinery creates bead gt-abc.1: "Resolve merge conflicts: gt-abc"
         Conflict metadata saved in bead description
         New MR created for conflict-resolution task
         Priority boosted (retry_count=1, ConvoyCreatedAt set)

T=1:50   Polecat-2 spawned for gt-abc.1 (conflict resolution)
T=3:10   Polecat-2 rebases and resolves conflicts
T=3:15   Polecat-2 submits new MR → state: recyclable

T=3:30   Refinery processes rebase MR → SUCCESS (clean merge)
         Original work (gt-abc) is now on main
         Conflict-resolution task (gt-abc.1) closes

T=3:31   Polecat-1 recycled (Witness cleanup)
T=3:32   Polecat-2 recycled (Witness cleanup)
```

## Conflict Metadata Structure

Conflict-resolution beads store metadata in their description:

```markdown
## Metadata
- Original MR: mr-1234-5abc
- Branch: polecat/Toast-xyz789
- Conflict with: main@abc12345def67890
- Original issue: gt-xyz
- Retry count: 1

## Details
The merge request encountered conflicts when attempting to merge
into main. This task is to rebase the branch and resolve conflicts.
```

This enables:
- Tracking of which MR had the conflict
- Recovery of the exact branch and target state
- Escalating priority as retries accumulate
- Links back to the original work

## Why Ephemeral?

Traditional models keep worker agents alive across multiple tasks:
- State accumulates
- Session memory grows
- Recovery from crashes requires state reconstruction
- Task dependencies become implicit

The ephemeral model:
- **Spawns fresh per assignment** (clean state every time)
- **Exits at MR submission** (not at merge completion)
- **Conflicts become new assignments** (not original polecat's problem)
- **Recyclable at exit** (no long-term session management)
- **Scalable** (polecats are cheap to create/destroy)

This aligns with the Propulsion Principle: each agent does one thing completely and cleanly hands off the result.

## Related Docs

- **refinery-workflow.md**: How the refinery processes merge requests and handles failures
- **polecat-wisp-architecture.md**: Molecule/wisp patterns for work execution
- **convoy.md**: Batch work coordination (related to convoy starvation prevention in scoring)
