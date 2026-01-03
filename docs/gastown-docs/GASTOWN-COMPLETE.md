# Gas Town Complete Reference

Compressed documentation for AI/LLM consumption.

---

# Understanding Gas Town

Orchestration layer treating AI agent work as structured data. Every action is attributed, every agent has a track record.

## Role Taxonomy

**Infrastructure:**
| Role | Description | Lifecycle |
|------|-------------|-----------|
| Mayor | Global coordinator | Singleton, persistent |
| Deacon | Background daemon | Singleton, persistent |
| Witness | Per-rig polecat manager | One per rig, persistent |
| Refinery | Per-rig merge queue | One per rig, persistent |

**Workers:**
| Role | Description | Lifecycle |
|------|-------------|-----------|
| Polecat | Ephemeral worker, own worktree | Transient, Witness-managed |
| Crew | Persistent worker, own clone | Long-lived, user-managed |
| Dog | Deacon helper | Ephemeral, Deacon-managed |

## Convoys: Tracking Work

```bash
gt convoy create "Feature X" gt-abc gt-def --notify overseer
gt convoy status hq-cv-abc
gt convoy list  # Dashboard
```

Convoy = persistent tracking. Swarm = ephemeral workers on convoy issues.

## Crew vs Polecats

| Aspect | Crew | Polecat |
|--------|------|---------|
| Lifecycle | User controls | Witness controls |
| Git state | Push to main | Branch → Refinery merges |
| Cleanup | Manual | Automatic |
| Use for | Exploratory, human judgment | Discrete, parallelizable tasks |

## Dogs vs Crew

Dogs are NOT workers. They're Deacon helpers for infrastructure (e.g., Boot triages Deacon health). Use Crew for project work.

## Cross-Rig Work

**Worktrees (preferred):** `gt worktree beads` creates `~/gt/beads/crew/gastown-joe/` - identity preserved.

**Dispatch:** `gt convoy create "Fix" bd-xyz && gt sling bd-xyz beads` - work owned by target rig.

## Identity

All work attributed:
```
Git:    gastown/crew/joe <owner@example.com>
Beads:  created_by: gastown/crew/joe
```

Identity preserved cross-rig.

## Propulsion Principle

> **If you find something on your hook, YOU RUN IT.**

No confirmation. Execute immediately.

## Model A/B Testing

```bash
gt sling gt-abc gastown --model=claude-sonnet
gt sling gt-def gastown --model=gpt-4
bd stats --actor=gastown/polecats/* --group-by=model
```

## Common Mistakes

1. Using dogs for user work (use crew/polecats)
2. Confusing crew with polecats (crew=persistent, polecats=transient)
3. Working in wrong directory (cwd = identity detection)
4. Waiting for confirmation when work hooked (hook IS assignment)

---

# Why These Features?

Enterprise AI challenges → Gas Town solutions.

## The Problem

AI agents write code, review PRs, fix bugs. Can't answer:
- Who did what? Which agent introduced this bug?
- Who's reliable? Which agents deliver quality?
- What's connected? Does this frontend depend on backend PR?

## The Solution: A Work Ledger

Work as structured data. Every action recorded. Every agent has a track record.

## Entity Tracking and Attribution

**Problem:** 50 agents, 10 projects, one bug. Which agent?
**Solution:** Distinct identity per agent.
```
Git: gastown/polecats/toast <owner@example.com>
Beads: created_by: gastown/crew/joe
```
**Why:** Debugging, compliance (SOX/GDPR), accountability.

## Work History (Agent CVs)

**Problem:** Complex Go refactor. 20 agents. Who's best?
**Solution:** Accumulated work history.
```bash
bd audit --actor=gastown/polecats/toast
bd stats --actor=gastown/polecats/toast --tag=go
```
**Why:** Performance management, capability matching, A/B testing models.

## Capability-Based Routing

**Problem:** Work in Go, Python, TypeScript. Manual assignment doesn't scale.
**Solution:** Skills from work history, automatic matching.
```bash
gt dispatch gt-xyz --prefer-skill=go
```

## Recursive Work Decomposition

**Problem:** Feature = 50 tasks across 8 repos. Flat lists don't capture structure.
**Solution:** Natural decomposition: Epic → Feature → Task. Roll-ups automatic.

## Cross-Project References

**Problem:** Frontend can't ship until backend lands. Different repos.
**Solution:** Explicit dependencies: `depends_on: beads://github/acme/backend/be-456`

## Federation

**Problem:** Multi-repo, multi-org visibility fragmented.
**Solution:** Federated workspaces.
```bash
gt remote add partner hop://partner.com/project
bd list --remote=partner
```

## Validation and Quality Gates

**Problem:** Agent says "done." Is it actually done?
**Solution:** Structured validation with attribution.
```json
{"validated_by": "gastown/refinery", "tests_passed": true}
```

## Real-Time Activity Feed

**Problem:** Multi-agent work is opaque.
**Solution:** `bd activity --follow` - work state as stream.

## Design Philosophy

1. Attribution not optional
2. Work is data (structured, queryable)
3. History matters (track records = trust)
4. Scale assumed (multi-repo, multi-agent, multi-org)
5. Verification over trust (quality gates are primitives)

---

# The Propulsion Principle

> **If you find something on your hook, YOU RUN IT.**

Gas Town is a steam engine. Agents are pistons. When work is hooked, EXECUTE.

## Why

- No supervisor polling "did you start yet?"
- The hook IS your assignment
- Every delay stalls the engine
- Other agents may be blocked on YOUR output

## The Contract

1. You will find it on your hook
2. You will understand it (`bd show` / `gt hook`)
3. You will BEGIN IMMEDIATELY

## Propulsion Loop

```bash
bd close gt-abc.3 --continue   # One command: close + auto-advance
```

**Old way (3 commands, momentum lost):**
```bash
bd close gt-abc.3
bd ready --parent=gt-abc
bd update gt-abc.4 --status=in_progress
```

**New way (1 command, momentum preserved):**
```bash
bd close gt-abc.3 --continue
```

## Startup Behavior

1. `gt hook` - check hook
2. Work hooked → EXECUTE immediately
3. Hook empty → Check mail
4. Nothing → ERROR: escalate to Witness

## The Capability Ledger

Every completion is recorded. Your work is visible.
- Redemption is real (consistent work builds over time)
- Every completion = evidence autonomous execution works
- Your CV grows with every completion

Execute with care.

---

# Installing Gas Town

## Prerequisites

| Required | Version | Check |
|----------|---------|-------|
| Go | 1.24+ | `go version` |
| Git | 2.20+ | `git --version` |
| Beads | latest | `bd version` |

**Optional (Full Stack):** tmux 3.0+, Claude Code

## Install

```bash
# macOS
brew install go git tmux

# Linux (Debian/Ubuntu)
sudo apt install -y git
# Go: use go.dev installer (apt may be outdated)
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
```

```bash
# Install binaries
go install github.com/steveyegge/gastown/cmd/gt@latest
go install github.com/steveyegge/beads/cmd/bd@latest

# Create workspace
gt install ~/gt

# Add a project
gt rig add myproject https://github.com/you/repo.git

# Verify
gt doctor
gt status
```

## Modes

**Minimal (No Daemon):** Manual Claude instances, Gas Town tracks state only
```bash
gt convoy create "Fix bugs" issue-123
cd ~/gt/myproject/polecats/<worker> && claude --resume
```

**Full Stack (Daemon):** Agents run in tmux, auto-managed
```bash
gt daemon start
gt convoy create "Feature X" issue-123
gt sling issue-123 myproject
gt mayor attach  # monitor
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `gt: command not found` | Add `$HOME/go/bin` to PATH |
| `bd: command not found` | `go install .../beads/cmd/bd@latest` |
| Doctor errors | `gt doctor --fix` |
| Daemon not starting | Check tmux: `tmux -V` |
| Git auth issues | Test: `ssh -T git@github.com` |
| Beads sync issues | `bd sync --status`, `bd doctor` |

## Update

```bash
go install github.com/steveyegge/gastown/cmd/gt@latest
go install github.com/steveyegge/beads/cmd/bd@latest
gt doctor --fix
```

---

# Agent Identity and Attribution

Every action, commit, and bead update is linked to a specific agent identity.

## BD_ACTOR Format

| Role | Format | Example |
|------|--------|---------|
| Mayor | `mayor` | `mayor` |
| Deacon | `deacon` | `deacon` |
| Witness | `{rig}/witness` | `gastown/witness` |
| Refinery | `{rig}/refinery` | `gastown/refinery` |
| Crew | `{rig}/crew/{name}` | `gastown/crew/joe` |
| Polecat | `{rig}/polecats/{name}` | `gastown/polecats/toast` |

## Attribution

**Git commits:**
```bash
GIT_AUTHOR_NAME="gastown/crew/joe"      # Agent (executor)
GIT_AUTHOR_EMAIL="steve@example.com"    # Owner (human)
# Result: abc123 Fix bug (gastown/crew/joe <steve@example.com>)
```

**Beads records:** `created_by`, `updated_by` from BD_ACTOR

**Events:** All events include `actor` field

## Environment (auto-set by daemon)

```bash
BD_ACTOR="gastown/polecats/toast"
GIT_AUTHOR_NAME="gastown/polecats/toast"
GT_ROLE="polecat"
GT_RIG="gastown"
GT_POLECAT="toast"
```

## Audit Queries

```bash
bd audit --actor=gastown/crew/joe    # All work by agent
bd audit --actor=gastown/*           # All work in rig
git log --author="gastown/crew/joe"  # Git history
```

## CV Model

```
steve@example.com          ← global identity (from git)
├── Town A                 ← workspace
│   └── gastown/crew/joe   ← agent executor
└── Town B
    └── acme/polecats/nux  ← agent executor
```

**Key points:**
- Global ID = email (already in git commits)
- Agents execute, humans own
- Polecats are ephemeral (like K8s pods)
- Skills derived from work evidence (files touched, tags, patterns)
- Multi-town aggregation via `bd cv email@example.com`

## Design Principles

1. Agents not anonymous - every action attributed
2. Work owned, not authored - agent creates, overseer owns
3. Attribution permanent - git preserves history
4. Format parseable - enables programmatic analysis
5. Consistent across systems - same format in git, beads, events

---

# Operational State

Runtime state: events (history) + labels (cache) + Boot (triage).

## Events: State Transitions as Beads

| Event Type | Description |
|------------|-------------|
| `patrol.muted` | Patrol disabled |
| `patrol.unmuted` | Patrol re-enabled |
| `agent.started/stopped` | Session lifecycle |
| `mode.degraded/normal` | System mode |

```bash
# Create event
bd create --type=event --event-type=patrol.muted \
  --actor=human:overseer --target=agent:deacon \
  --payload='{"reason":"fixing deadlock"}'

# Query events
bd list --type=event --target=agent:deacon
```

## Labels-as-State

Events = full history. Labels = current state cache.

Format: `<dimension>:<value>` (e.g., `patrol:muted`, `mode:degraded`)

```bash
# State change flow
bd create --type=event --event-type=patrol.muted ...
bd update role-deacon --add-label=patrol:muted --remove-label=patrol:active

# Query current state
bd list --type=role --label=patrol:muted
```

## Boot: Deacon's Watchdog

Boot triages Deacon health. Daemon spawns Boot fresh each tick.

**Lifecycle:**
```
Daemon tick → Check marker → Spawn Boot → Triage molecule:
  Observe (wisps, mail, git, tmux) → Decide → Act → Clean inbox → Exit
```

**Decision options (increasing disruption):**
- Let continue (if progress evident)
- `gt nudge <agent>` (gentle wake)
- Escape + chat (interrupt)
- Request restart (last resort)

```bash
gt dog status boot
gt dog call boot      # Manual run
gt dog prime boot     # Prime context
```

## Degraded Mode

Without tmux, reduced capabilities.

```bash
GT_DEGRADED=true  # Set by daemon
```

| Capability | Normal | Degraded |
|------------|--------|----------|
| Observe panes | Yes | No |
| Interactive interrupt | Yes | No |
| Agent spawn | tmux | Direct |
| Boot lifecycle | Handoff | Exit |

## Config vs State

| Type | Storage |
|------|---------|
| Static config | TOML files |
| Operational state | Beads (events + labels) |
| Runtime flags | Marker files |

*Events are truth. Labels are cache.*

---

# Molecules

Workflow templates for multi-step work.

## Lifecycle

```
Formula (TOML) ─── bd cook ───► Protomolecule (frozen)
                                    │
                    ├── bd mol pour ───► Molecule (persistent) ─► bd squash ─► Digest
                    └── bd mol wisp ───► Wisp (ephemeral) ──────► bd burn ─► (gone)
```

| Term | Description |
|------|-------------|
| Formula | Source TOML template |
| Protomolecule | Frozen, ready to instantiate |
| Molecule | Active workflow with trackable steps |
| Wisp | Ephemeral molecule (never synced) |
| Digest | Squashed summary of completed mol |

## Navigation

```bash
bd mol current              # Where am I?
bd mol current gt-abc       # Status of specific mol
```

Output:
```
  ✓ gt-abc.1: Design
  ✓ gt-abc.2: Scaffold
  → gt-abc.3: Implement [in_progress] <- YOU ARE HERE
  ○ gt-abc.4: Tests
```

## Transitions

```bash
bd close gt-abc.3 --continue   # Close and auto-advance to next
bd close gt-abc.3 --no-auto    # Close without advancing
```

## Commands

**Beads (data):**
```bash
bd formula list / show / cook
bd mol list / show / pour / wisp / bond / squash / burn / current
```

**Agent (execution):**
```bash
gt hook                  # What's on my hook
gt mol current           # What to work on next
gt mol attach / detach   # Pin/unpin molecule
gt mol step done <step>  # Complete step
```

## Best Practices

1. Use `--continue` for momentum
2. Check `bd mol current` before resuming
3. Squash completed molecules for audit trail
4. Burn routine wisps

---

# Polecat Wisp Architecture

Polecats execute work via hooked molecules, closing steps sequentially.

## Molecule Types

| Type | Storage | Use |
|------|---------|-----|
| Molecule | `.beads/` (synced) | Discrete deliverables, audit trail |
| Wisp | `.beads-wisp/` (ephemeral) | Patrol cycles, no accumulation |

Polecats use **molecules** (audit value). Patrol agents use **wisps**.

## Step Execution

```bash
# Propulsion approach (1 command)
bd mol current                    # Where am I?
# ... do the work ...
bd close gt-abc.3 --continue      # Close and auto-advance
```

The `--continue` flag: closes step → finds next ready → marks in_progress → outputs transition.

## Example Session

```
$ bd mol current
  ✓ gt-abc.1: Design
  → gt-abc.2: Implement [in_progress] <- YOU ARE HERE
  ○ gt-abc.3: Tests

$ bd close gt-abc.2 --continue
✓ Closed gt-abc.2: Implement
→ Marked gt-abc.3: Tests in_progress
```

## Molecule Completion

```bash
# After last step
bd mol squash gt-abc --summary "Implemented user auth"
# Or for routine: bd mol burn gt-abc
```

## Hook Management

```bash
gt hook                        # What's on my hook?
gt mail inbox                  # Check for assignments
gt mol attach-from-mail <id>   # Attach work from mail
gt done                        # Complete: sync, merge queue, notify Witness
```

## Workflow Summary

```
1. Spawn with work on hook
2. gt hook → bd mol current → Execute step
3. bd close <step> --continue
4. If more steps: GOTO 2
5. gt done
6. Wait for Witness cleanup
```

## Wisp vs Molecule

| Question | Molecule | Wisp |
|----------|----------|------|
| Audit trail needed? | Yes | No |
| Repeats continuously? | No | Yes |
| Discrete deliverable? | Yes | No |

---

# Convoys

Persistent tracking units for batched work across rigs.

## Commands

```bash
gt convoy create "Feature X" gt-abc gt-def --notify overseer  # Create
gt convoy status hq-cv-abc                                     # Check progress
gt convoy list                                                 # Dashboard (active)
gt convoy list --all                                           # Include landed
```

## Concept

```
            🚚 Convoy (hq-cv-abc)
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
   [gt-xyz]    [gt-def]    [bd-abc]
       │            │            │
       ▼            ▼            ▼
   [polecat]   [polecat]   [polecat]
              "the swarm"
```

| Concept | Persistent | Description |
|---------|------------|-------------|
| **Convoy** | Yes (hq-cv-*) | Tracking unit you create/monitor |
| **Swarm** | No | Ephemeral workers on convoy issues |

## Lifecycle

```
OPEN ──(all issues close)──► CLOSED
  ↑                              │
  └──(add issues: auto-reopens)──┘
```

## Usage

**Create:**
```bash
gt convoy create "Deploy v2.0" gt-abc bd-xyz --notify gastown/joe
gt convoy create "Fix auth" gt-auth-fix  # Single issue OK
```

**Add issues** (direct bd command):
```bash
bd dep add hq-cv-abc gt-new-issue --type=tracks
```

**Status output:**
```
🚚 hq-cv-abc: Deploy v2.0
  Status:    ●
  Progress:  2/4 completed
  ✓ gt-xyz: Update API endpoint
  ✓ bd-abc: Fix validation
  ○ bd-ghi: Update docs
```

## Auto-Convoy

`gt sling bd-xyz beads/amber` auto-creates convoy "Work: bd-xyz" for dashboard visibility.

## Cross-Rig Tracking

Convoys (hq-cv-*) track issues from any rig. The `tracks` relation is non-blocking and additive.

```bash
gt convoy create "Full-stack" gt-frontend-abc gt-backend-def bd-docs-xyz
```

## Notifications

```bash
gt convoy create "X" gt-abc --notify gastown/joe --notify mayor/
```

Landing notification includes: issue list, duration, completion status.

## Convoy vs Rig Status

| `gt convoy status` | Cross-rig batch progress |
| `gt rig status` | Single rig worker activity |

---

# Swarm (Ephemeral Worker View)

"Swarm" = workers currently assigned to a convoy's issues. No ID, no persistent state.

| Concept | Persistent | Description |
|---------|------------|-------------|
| Convoy | Yes (hq-*) | Tracking unit you create |
| Swarm | No | Ephemeral view of workers |

## Relationship

```
Convoy → tracks → Issues → assigned to → Polecats ("the swarm")
```

"Kick off a swarm" = create convoy + assign polecats. Work completes → convoy lands → swarm dissolves.

## Viewing

```bash
gt convoy status hq-abc
```

```
Progress: 2/3 complete
Issues:
  ✓ gt-xyz: Update API
  → bd-ghi: Docs          @beads/amber

Workers (the swarm):
  beads/amber   bd-ghi   running 12m
```

## Note

`gt swarm` deprecated in favor of `gt convoy`. Convoy is persistent tracking; swarm is ephemeral workers.

---

# Mail Protocol

Inter-agent communication via `type=message` beads routed by `gt mail`.

## Message Types

| Type | Route | Purpose |
|------|-------|---------|
| `POLECAT_DONE` | Polecat → Witness | Signal completion, trigger cleanup |
| `MERGE_READY` | Witness → Refinery | Branch ready for merge queue |
| `MERGED` | Refinery → Witness | Merge success, safe to nuke polecat |
| `MERGE_FAILED` | Refinery → Witness | Non-conflict failure (tests/build) |
| `REWORK_REQUEST` | Refinery → Witness | Conflicts, need rebase |
| `WITNESS_PING` | Witness → Deacon | Second-order monitoring |
| `HELP` | Any → Mayor | Request intervention |
| `HANDOFF` | Agent → self | Session continuity |

## Body Formats

**POLECAT_DONE:**
```
Exit: MERGED|ESCALATED|DEFERRED
Issue: <issue-id>
MR: <mr-id>
Branch: <branch>
```

**MERGE_READY/MERGED:**
```
Branch: <branch>
Issue: <issue-id>
Polecat: <polecat-name>
```

**HANDOFF:**
```
attached_molecule: <mol-id>
## Context
<notes>
## Next
<what to do>
```

## Flows

**Completion:** Polecat → POLECAT_DONE → Witness (verify) → MERGE_READY → Refinery (merge) → MERGED → Witness (nuke)

**Failure:** Refinery → MERGE_FAILED → Witness → Polecat (rework)

**Conflict:** Refinery → REWORK_REQUEST → Witness → Polecat (rebase) → POLECAT_DONE → retry

## Addresses

```
greenplace/witness           # Rig role
greenplace/polecats/nux      # Specific polecat
mayor/                       # Town-level
```

## Commands

```bash
gt mail send <addr> -s "Subject" -m "Body"
gt mail inbox
gt mail read <id>
gt mail ack <id>
```

## Conventions

- Subject: `TYPE_NAME <context>` (uppercase prefix)
- Body: Key-value pairs, blank line, then freeform markdown

---

# Escalation Protocol

Agents escalate when automated resolution fails. Routes through tiers: Worker → Deacon → Mayor → Overseer.

## Severity Levels

| Level | Priority | Description |
|-------|----------|-------------|
| CRITICAL | P0 | System-threatening (data corruption, security breach) |
| HIGH | P1 | Important blocker (merge conflict, critical bug) |
| MEDIUM | P2 | Standard (design decision, unclear requirements) |

## Categories

| Category | Route | Use Case |
|----------|-------|----------|
| `decision` | Deacon → Mayor | Multiple valid paths |
| `help` | Deacon → Mayor | Need guidance |
| `blocked` | Mayor | Unresolvable dependency |
| `failed` | Deacon | Unexpected error |
| `emergency` | Overseer (direct) | Security/data integrity |
| `gate_timeout` | Deacon | Gate didn't resolve |
| `lifecycle` | Witness | Worker stuck |

## Commands

```bash
# Basic
gt escalate "Database migration failed"
gt escalate -s CRITICAL "Data corruption detected"

# Category-based
gt escalate --type decision "Which auth approach?"
gt escalate --type emergency "Security vulnerability"

# Explicit routing
gt escalate --to mayor "Cross-rig coordination"
gt escalate --forward --to mayor "Deacon couldn't resolve"

# Structured decision
gt escalate --type decision \
  --question "Which auth approach?" \
  --options "JWT,Session cookies,OAuth2" \
  --context "Admin panel needs login" \
  --issue bd-xyz
```

## Tiered Flow

```
Worker → gt escalate
    ↓
Deacon ─► Can resolve? → Update issue, re-sling
    │
    └► Cannot? → gt escalate --forward --to mayor
                     ↓
                Mayor → Same pattern → Overseer
```

## Decision Pattern

`--type decision` updates issue with structured format:

```markdown
## Decision Needed
**Question:** Which auth approach?
| Option | Description |
| A | JWT tokens |
| B | Session cookies |
**To resolve:** Comment with choice, reassign to worker
```

## Integration

- **Gate timeouts**: Witness escalates when timer gates expire
- **Stuck polecats**: Witness escalates lifecycle issues
- **Merge failures**: Refinery escalates failed merges

## Polecat Exit Pattern

```bash
bd update $ISSUE --notes "## Decision Needed..."
gt escalate --type decision --issue $ISSUE "Needs decision"
gt done --exit ESCALATED
```

## When to Escalate

**Do:** System errors, security issues, unresolvable conflicts, ambiguous requirements, design decisions, stuck loops, gate timeouts

**Don't:** Normal workflow, recoverable errors, answerable from context

## Viewing

```bash
bd list --status=open --tag=escalation
bd close <id> --reason "Resolved by fixing X"
```

---

# Wisp Squash Design

How wisps condense to digests for audit trail and activity feed.

## Squash Cadences

**Patrol Wisps (Deacon, Witness, Refinery):**
- Trigger: End of each patrol cycle
- `loop-or-exit` step → squash current wisp → new wisp or handoff

**Work Wisps (Polecats):**
- Trigger: Before `gt done` or molecule completion
- Complete → squash to digest | Abandoned → burn | Handed off → squash with context

## Summary Templates

```markdown
## Patrol Digest: {{.Agent}}
**Cycle**: {{.CycleNumber}} | **Duration**: {{.Duration}}
### Actions Taken
{{range .Actions}}- {{.Description}}{{end}}
### Metrics
- Inbox: {{.InboxCount}} | Alerts: {{.AlertCount}}
```

**Formula-defined templates:**
```toml
[squash]
template = "..."
[squash.vars]
summary_length = "short"
```

Resolution: Formula template → Type default → Minimal fallback.

## Retention Rules

```
Wisp → Squash → Digest (active: 30d) → Archived (1yr) → Rollup (permanent)
```

**Weekly/Monthly Rollups:** Aggregate stats preserved permanently.

Config:
```json
{"retention": {"digest_active_days": 30, "rollup_weekly": true}}
```

## Commands

```bash
bd mol squash <id>                    # Use formula template
bd mol squash <id> --template=detailed
bd list --label=digest                # View digests
bd archive --older-than=30d           # Archive old
bd rollup generate --week=2025-01     # Generate rollup
```

## Activity Feed Integration

```json
{"type": "digest", "agent": "witness", "summary": "Patrol 47 complete", "metrics": {...}}
```

## Design Decisions

- Squash per-cycle (not session-end) - finer granularity, crash resilience
- Formula-defined templates - molecule-specific summaries
- Retention tiers - balance detail vs storage
- Rollups preserve trends - long-term insights without noise

---

# Federation Architecture

Multi-workspace coordination for Gas Town and Beads.

## Why Federation

Real projects span multiple repos: microservices, platform teams, contractors, acquisitions. Federation provides unified tracking while preserving team autonomy.

## Entity Model

```
Level 1: Entity    - Person or organization
Level 2: Chain     - Workspace/town per entity
Level 3: Work Unit - Issues, tasks, molecules
```

## URI Scheme

```
hop://entity/chain/rig/issue-id           # Full HOP reference
hop://steve@example.com/main-town/gp/gp-xyz

beads://platform/org/repo/issue-id        # Cross-repo (same platform)
beads://github/acme/backend/ac-123

gp-xyz                                     # Local (prefix routes)
greenplace/gp-xyz                          # Different rig, same chain
```

## Relationship Types

**Employment:**
```json
{"type": "employment", "entity": "alice@example.com", "organization": "acme.com"}
```

**Cross-Reference:**
```json
{"references": [{"type": "depends_on", "target": "hop://other/chain/rig/id"}]}
```

**Delegation:**
```json
{"type": "delegation", "parent": "hop://acme/proj-123", "child": "hop://alice/gp-xyz"}
```

## Agent Provenance

```bash
# Git commits
GIT_AUTHOR_NAME="greenplace/crew/joe"
GIT_AUTHOR_EMAIL="steve@example.com"  # Workspace owner

# Beads operations
BD_ACTOR="greenplace/crew/joe"
```

## Discovery

```bash
# Workspace metadata: ~/gt/.town.json
gt remote add acme hop://acme.com/engineering
bd show hop://acme.com/eng/ac-123    # Fetch remote issue
bd list --org=acme.com               # All org work
bd audit --actor=greenplace/crew/joe # Agent history
```

## Implementation Status

- [x] Agent identity in git commits
- [x] BD_ACTOR default in beads create
- [x] Workspace metadata (.town.json)
- [x] Cross-workspace URI scheme
- [ ] Remote registration
- [ ] Cross-workspace queries
- [ ] Delegation primitives

## Design Principles

1. Flat namespace - entities not nested, relationships connect them
2. Relationships over hierarchy - graph structure, not tree
3. Git-native - uses git mechanics (remotes, refs)
4. Incremental - works standalone, gains power with federation
5. Privacy-preserving - each entity controls visibility

---

# Towers of Hanoi Demo

Durability proof: Gas Town executes arbitrarily long sequential workflows with crash recovery.

## What It Proves

- Large molecule creation (1000+ issues)
- Sequential execution with chained dependencies
- Crash recovery (resume after restart)
- Nondeterministic idempotence

## Scale

| Disks | Moves | Formula | Runtime |
|-------|-------|---------|---------|
| 7 | 127 | ~19 KB | ~14 sec |
| 10 | 1,023 | ~149 KB | ~2 min |
| 15 | 32,767 | ~4.7 MB | ~1 hour |

## Running

```bash
# Create wisp
WISP=$(bd mol wisp towers-of-hanoi-10 --json | jq -r '.new_epic_id')

# Get child IDs
bd list --parent=$WISP --limit=2000 --json | jq -r '.[].id' > /tmp/ids.txt

# Execute (serial close)
while read id; do bd --no-daemon close "$id" >/dev/null 2>&1; done < /tmp/ids.txt

# Cleanup
bd mol burn $WISP --force
```

## Why Wisps

- No git pollution (don't sync to JSONL)
- Auto-cleanup (burn without tombstones)
- Speed (no export overhead)
- `bd ready` excludes wisps by design

## Key Notes

- Dependencies via `needs` create proper `blocks` relations
- Close speed: ~109ms each, ~9/second serial
- Parent-child is hierarchy only, doesn't block execution

## Session Cycling

```bash
# Session 1
WISP=$(bd mol wisp towers-of-hanoi-10 --json | jq -r '.new_epic_id')
gt handoff -s "Hanoi demo" -m "Wisp: $WISP, progress: 400/1025"

# Session 2
bd list --parent=$WISP --status=open --limit=2000 --json | jq -r '.[].id' > /tmp/ids.txt
# Continue closing...
```

The molecule IS the state. No session memory needed.

---

# Gas Town Reference

## Directory Structure

```
~/gt/                       Town root
├── .beads/                 Town-level beads (hq-*)
├── mayor/town.json
└── <rig>/                  Project container (NOT git clone)
    ├── config.json         Rig identity
    ├── .repo.git/          Bare repo (shared by worktrees)
    ├── mayor/rig/          Mayor clone (canonical beads)
    ├── refinery/rig/       Worktree on main
    ├── witness/            No clone (monitors only)
    ├── crew/<name>/        Human workspaces
    └── polecats/<name>/    Worker worktrees
```

## Beads Routing

```bash
bd show gp-xyz    # Routes to greenplace rig
bd show hq-abc    # Routes to town-level
# Debug: BD_DEBUG_ROUTING=1 bd show <id>
```

## Configuration

**Rig** (`config.json`): `{name, git_url, beads: {prefix}}`
**Settings** (`settings/config.json`): theme, max_workers, merge_queue
**Runtime** (`.runtime/`): PIDs, ephemeral (gitignored)

## Formula Format

```toml
formula = "name"
type = "workflow"
[[steps]]
id = "step-id"
title = "{{var}}"
needs = ["dependency"]
```

## Molecule Lifecycle

```
Formula → bd cook → Proto → bd mol pour → Mol → bd squash → Digest
                         └→ bd mol wisp → Wisp → bd burn → (gone)
```

## Key Commands

**Town:** `gt install`, `gt doctor [--fix]`
**Rig:** `gt rig add/list/remove`
**Convoy:** `gt convoy list/status/create` (primary dashboard)
**Work:** `gt sling <bead> <rig>` (auto-creates convoy)
**Mail:** `gt mail inbox/read/send`
**Escalation:** `gt escalate [-s CRITICAL|HIGH|MEDIUM] "msg"`
**Sessions:** `gt handoff`, `gt peek`, `gt nudge`, `gt seance`

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `BD_ACTOR` | Agent identity |
| `BEADS_NO_DAEMON` | Required for worktree polecats |
| `GT_ROLE` | Agent role (mayor, polecat, etc.) |
| `GT_RIG` | Rig name |

## Patrol Agents

| Agent | Molecule | Job |
|-------|----------|-----|
| Deacon | mol-deacon-patrol | Lifecycle, health |
| Witness | mol-witness-patrol | Monitor polecats |
| Refinery | mol-refinery-patrol | Merge queue |

## Architecture

- Bare repo pattern: `.repo.git/` shared by worktrees
- Beads as control plane: molecule steps ARE beads issues
- Nondeterministic idempotence: any worker can continue any molecule
- Convoy tracking: persistent. Swarm: ephemeral workers on convoy issues
