# Refinery Workflow

## Overview

The **Refinery** is the merge queue processor for a rig. It monitors for merge requests from polecats, automatically attempts to merge them into target branches, and handles failures (conflicts, test failures, etc.).

The refinery is **mechanical and deterministic**: it follows a fixed algorithm without human judgment. All decisions are made by algorithms (priority scoring, conflict detection, failure categorization).

## Role in the Architecture

```
Polecat
  ↓
(submits MR to queue)
  ↓
Refinery
  ├─ Processes queue (priority-ordered)
  ├─ Attempts merge (git rebase + merge)
  ├─ Runs tests
  ├─ Handles failures
  │  ├─ Conflict → Create resolution task
  │  ├─ Test failure → Create fix task
  │  └─ Push fail → Retry with backoff
  └─ Closes MR (merged/rejected/conflict)
```

## Refinery States

The refinery itself has operational states:

| State | Meaning |
|-------|---------|
| `stopped` | Not running |
| `running` | Actively processing queue |
| `paused` | Temporarily suspended (manual or automatic) |

## Merge Request Lifecycle

Each MR moves through states as it's processed:

```
open (waiting)
  ↓
in_progress (being merged)
  ├─ Success → closed (merged)
  ├─ Conflict → closed (conflict)
  ├─ Test fail → closed (needs-fix)
  └─ Other error → open (retry) or closed (rejected)
```

### MR Status Values

| Status | Meaning | Can Transition To |
|--------|---------|-------------------|
| `open` | Waiting to be processed or needs rework | `in_progress`, `closed` |
| `in_progress` | Currently being merged by refinery | `closed`, `open` |
| `closed` | Processing complete (merged/rejected/conflict) | (immutable) |

### Close Reasons

When an MR is closed, a reason is recorded:

| Reason | Meaning | Action |
|--------|---------|--------|
| `merged` | Successfully merged to target | Polecat recyclable, original work complete |
| `conflict` | Merge conflicts with target | Create conflict-resolution task |
| `rejected` | Manual rejection | Polecat recyclable, work rejected |
| `superseded` | Replaced by another MR | Polecat recyclable, work replaced |

## Merge Process

### Step 1: Claim

A refinery worker claims an MR from the queue:
1. Select the highest-priority unclaimed MR
2. Attempt to claim it (atomic operation)
3. If already claimed by another worker, skip to next

Claims expire after 10 minutes (crash recovery).

```bash
gt refinery claim <mr-id>
```

### Step 2: Fetch and Checkout

```bash
git fetch origin <source-branch>
git checkout <target-branch>
git pull origin <target-branch>
git fetch origin <source-branch>
```

Failures here are retried with exponential backoff (default: 3 retries, 1s→2s→4s).

### Step 3: Rebase and Merge

The refinery attempts a **rebase-style merge**:

```bash
git merge --no-ff <source-branch>  # Explicit merge commit
```

Possible outcomes:
1. **Success**: Clean merge, no conflicts → proceed to step 4
2. **Conflict**: Git merge fails → conflict detection (step 6)
3. **Error**: Unexpected git failure → release MR, retry later

### Step 4: Test

If configured, run tests on the merged state:

```bash
go test ./...  # (or configured test command)
```

Outcomes:
- **Pass**: Merge succeeds → proceed to step 5
- **Fail**: Test failure detected → failure handling (step 6)
- **Flaky**: Test might be intermittent → configurable retry policy

### Step 5: Push and Complete

```bash
git push origin <target-branch>
```

Retry with exponential backoff if push fails.

On success:
1. **Delete source branch** (configurable): `git push origin --delete <source-branch>`
2. **Close MR**: status=`closed`, reason=`merged`
3. **Polecat becomes recyclable** (Witness cleans up)

### Step 6: Failure Handling

If merge, test, or push fails:

#### Conflict Detection

When git merge fails with conflicts:
1. **Identify conflict markers** (git status, conflict files)
2. **Record conflict state**:
   - `TargetBranch@SHA`: What it conflicted with
   - `SourceBranch`: What needs rebasing
   - `MRCreatedAt`: Original submission time

#### Create Conflict-Resolution Task

A new bead is created in the rig:

```
Title: Resolve merge conflicts: <original-issue>
Description:
  ## Metadata
  - Original MR: mr-1234-5abc
  - Branch: polecat/Toast-xyz789
  - Conflict with: main@abc12345
  - Original issue: gt-xyz
  - Retry count: 1
```

The new task is:
1. **Assigned high priority** (retry count boosts score)
2. **Queued as a new MR** for the conflict-resolution work
3. **Isolated from original polecat** (separate work item)

#### Test Failures

If tests fail:
1. **Categorize failure** (flaky test vs. real failure)
2. **If real failure**: Create fix task (similar to conflict)
3. **If flaky**: Retry with limit (2-3 attempts)

#### Push Failures

If push to remote fails:
1. **Release the MR claim** (allow another worker)
2. **Reopen MR**: status=`open` (ready for retry)
3. **Retry on next cycle** (with exponential backoff limit)

## Queue Processing Algorithm

### Priority Ordering

MRs are processed in order of **priority score** (see ephemeral-polecat-workflow.md for scoring details).

```bash
gt refinery queue  # Shows queue ordered by score
```

### Processing Loop

```
1. List all unclaimed MRs (sorted by score)
2. For each MR:
   a. Attempt to claim
   b. If claimed:
      - Perform merge process (steps 1-6 above)
      - Update MR status
   c. Release claim if failed
3. Sleep (configurable interval, default: 5s)
4. Loop back to step 1
```

### Parallel Workers

Multiple refinery workers can process the same queue in parallel:

```bash
GT_REFINERY_WORKER=refinery-1 gt refinery start rig &
GT_REFINERY_WORKER=refinery-2 gt refinery start rig &
```

Each worker:
1. Claims one MR at a time
2. Processes it independently
3. Updates queue state atomically
4. Moves to next unclaimed MR

Atomic claiming prevents double-processing.

## Configuration

Merge behavior is configured via `MergeConfig`:

```json
{
  "run_tests": true,
  "test_command": "go test ./...",
  "delete_merged_branches": true,
  "push_retry_count": 3,
  "push_retry_delay_ms": 1000
}
```

| Setting | Default | Purpose |
|---------|---------|---------|
| `run_tests` | true | Run tests after merge |
| `test_command` | "go test ./..." | Command to run for testing |
| `delete_merged_branches` | true | Delete source branch after merge |
| `push_retry_count` | 3 | Retries for push failures |
| `push_retry_delay_ms` | 1000 | Base delay (exponential backoff) |

## Conflict Metadata

When conflicts are detected, metadata is saved in the conflict-resolution task:

| Field | Purpose |
|-------|---------|
| `OriginalMRID` | Reference to the failed MR (traceability) |
| `Branch` | Source branch needing rebase |
| `TargetBranch` | Target branch (usually "main") |
| `ConflictSHA` | SHA of target when conflict was detected (for debugging) |
| `SourceIssue` | Original work issue (links back to root cause) |
| `RetryCount` | How many times this MR has been retried |

This metadata enables:
- **Debugging**: Understand what conflicted with what
- **Recovery**: Replay the rebase exactly
- **Priority**: Escalate conflicts that keep failing
- **Audit**: Track work history across conflict cycles

## Example: Processing a Queue

```
Queue (sorted by score):
  1. mr-1234-5abc (score: 1480, convoy P4 48h)    ← Process first
  2. mr-1234-5def (score: 1400, P0 standalone)
  3. mr-1234-5ghi (score: 1350, P1, 1 retry)

Worker 1 claims mr-1234-5abc:
  ✓ Fetch source branch
  ✓ Checkout main
  ✓ Merge
  ✗ Conflict detected!
  - Create conflict-resolution task (gt-abc.1)
  - New MR submitted for conflict-resolution
  - mr-1234-5abc closed (reason: conflict)

Worker 1 claims mr-1234-5def:
  ✓ Fetch
  ✓ Checkout
  ✓ Merge
  ✓ Tests pass
  ✓ Push
  - Source branch deleted
  - mr-1234-5def closed (reason: merged)
  - Original polecat recyclable

Worker 1 claims mr-1234-5ghi:
  ✓ Fetch
  ✓ Checkout
  ✓ Merge
  ✓ Tests pass
  ✗ Push fails (temporary network issue)
  - Release MR claim
  - mr-1234-5ghi reopened (retry on next cycle)

Next cycle:
  Worker 2 claims conflict-resolution MR
  Worker 1 claims mr-1234-5ghi (push retry)
  ...
```

## Workflow Commands

### Manage Refinery

```bash
# Start refinery for a rig
gt refinery start greenplace

# Run in foreground (for debugging)
gt refinery start greenplace --foreground

# Stop refinery
gt refinery stop greenplace

# Restart
gt refinery restart greenplace

# Attach to running session
gt refinery attach greenplace
```

### Monitor Queue

```bash
# Show refinery status
gt refinery status

# Show merge queue
gt refinery queue

# JSON output for automation
gt refinery queue --json

# Show unclaimed MRs
gt refinery unclaimed
```

### Manual Intervention

```bash
# Claim an MR (for testing)
gt refinery claim mr-1234-5abc

# Release an MR back to queue
gt refinery release mr-1234-5abc
```

## Design Principles

1. **Mechanical**: Follow algorithm, no human judgment in merge decisions
2. **Deterministic**: Same queue state always produces same processing order
3. **Atomic**: MR claims are atomic, preventing double-processing
4. **Scalable**: Multiple workers process in parallel
5. **Fault-tolerant**: Stale claims expire, crashed workers' work is recovered
6. **Conflict-agnostic**: Conflicts are work items, not blockers
7. **Observable**: Queue state is queryable and trackable

## Related Docs

- **ephemeral-polecat-workflow.md**: How polecats submit MRs and handle conflict-resolution
- **convoy.md**: Batch work coordination (convoys influence priority scoring)
- **priority.go**: Priority scoring formula implementation details
