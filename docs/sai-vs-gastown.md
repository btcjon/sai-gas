# SAI vs Gastown: Deep Comparative Analysis

> Last updated: 2026-01-02 | Status: Active | Author: Gemini CLI + Claude Opus

## Executive Summary

Steve Yegge's **Gastown** (December 2025) is a Go-based orchestrator for running 20-30+ Claude Code instances simultaneously. Our **SAI** system evolved independently over similar timeframes, solving overlapping problems with different approaches. This analysis identifies what Gastown solved elegantly that SAI still struggles with, and recommends specific adoptions.

**Bottom line**: Gastown is more opinionated and cohesive, optimizing for *completion* with a "factory" model. SAI is more flexible and fragmented, optimizing for *context preservation* with a "workshop" model. Gastown has better primitives for autonomous operation (GUPP, Molecules); SAI has better multi-model support and hook extensibility.

**Key Insight**: These systems operate at **different architectural layers** and are potentially **complementary, not competitive**.

---

## 0. Architectural Layers (Critical Understanding)

```
┌─────────────────────────────────────────────────────────────────┐
│                    ORCHESTRATION LAYER                          │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      GASTOWN (Go)                         │  │
│  │  • tmux session management                                │  │
│  │  • Beads state (JSONL/Git)                               │  │
│  │  • MEOW workflows (Molecules)                            │  │
│  │  • Worker supervision (Witness, Refinery)                │  │
│  │  • GUPP propulsion                                       │  │
│  └─────────────────────────┬─────────────────────────────────┘  │
│                            │ spawns                             │
│                            ▼                                    │
├─────────────────────────────────────────────────────────────────┤
│                    CLAUDE CODE LAYER                            │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      SAI (TypeScript)                     │  │
│  │  • Hooks (7 event types)                                 │  │
│  │  • Skills (159+ specialized contexts)                    │  │
│  │  • Agents (18 Task subagent types)                       │  │
│  │  • Rules (behavioral constraints)                        │  │
│  │  • mem0 (persistent memory)                              │  │
│  │  • Multi-model routing (Gemini, Codex, ZAI)             │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │ delegates to                       │
│                            ▼                                    │
├─────────────────────────────────────────────────────────────────┤
│                      WORKER LAYER                               │
│  • CLI tools (gemini, codex, zcc)                              │
│  • Task subagents (Opus/Haiku)                                 │
│  • External APIs                                                │
└─────────────────────────────────────────────────────────────────┘
```

### The Key Realization

| System | Layer | Controls | Doesn't Control |
|--------|-------|----------|-----------------|
| **Gastown** | Orchestration | tmux, workers, merges, supervision | What happens *inside* CC |
| **SAI** | CC Enhancement | Hooks, skills, memory, routing | What happens *outside* CC |

**They can coexist.** Gastown spawns CC instances via tmux. Each CC instance can run our SAI hooks/skills/agents. The integration point is Gastown's `.claude/` configuration (AGENTS.md, settings).

### Gastown's CC Configuration

From AGENTS.md, Gastown configures CC instances with:
- Mandatory git push before session end ("Work is NOT complete until `git push` succeeds")
- Quality gates (tests, lints, builds)
- Issue tracking workflow
- bd sync integration

**This is minimal.** Gastown relies on GUPP prompting + external supervision (Witness) rather than CC-level hooks. Our SAI hooks/skills/agents would **enhance** each Gastown-spawned CC instance.

---

## 1. Architecture Overview

### Gastown Architecture

```
+------------------------------------------------------------------+
|                        GASTOWN                                   |
+------------------------------------------------------------------+
|  Town (HQ)                                                       |
|  |-- Mayor (concierge, coordination)                             |
|  |-- Deacon (daemon, lifecycle, heartbeats)                      |
|  +-- Dogs (Deacon's helpers)                                     |
|                                                                  |
|  Per-Rig (per project):                                          |
|  |-- Witness (monitors polecats, unsticks them)                  |
|  |-- Refinery (merge queue)                                      |
|  |-- Polecats (ephemeral workers - worktrees)                    |
|  +-- Crew (persistent workers - clones)                          |
|                                                                  |
|  State: Beads (Git-backed JSONL)                                 |
|  UI: tmux                                                        |
|  Workflows: Formulas -> Protomolecules -> Molecules/Wisps        |
+------------------------------------------------------------------+
```

### SAI Architecture

```
+------------------------------------------------------------------+
|                          SAI                                     |
+------------------------------------------------------------------+
|  Claude Code (Main)                                              |
|  |-- Hooks (7 event types, TypeScript)                           |
|  |-- Skills (159+ specialized contexts)                          |
|  |-- Agents (18 specialized Task agents)                         |
|  +-- Rules (12 enforced patterns)                                |
|                                                                  |
|  Workers:                                                        |
|  |-- Task subagents (Opus/Haiku)                                 |
|  |-- CLI workers (gemini, codex, zcc)                            |
|  +-- Skill invocations                                           |
|                                                                  |
|  State: Ledgers (Markdown), mem0 (vector DB)                     |
|  UI: Zellij (panes), cc-wrapper                                  |
|  Workflows: TodoWrite + Ralph loops                              |
|                                                                  |
|  Automation:                                                     |
|  |-- manager.py (job scheduler)                                  |
|  |-- Ralph loops (verification)                                  |
|  +-- Watcher (improvement suggestions)                           |
+------------------------------------------------------------------+
```

---

## 2. Core Primitives Comparison

### 2.1 State Storage

| Aspect | Gastown (Beads) | SAI (Ledgers) |
|--------|-----------------|---------------|
| **Format** | JSONL + SQLite | Markdown |
| **Backing** | Git (synced) | Files (not synced) |
| **Queryable** | Yes (`bd show`, `bd ready`) | No (read entire file) |
| **Cross-rig** | Prefix routing (gt-*, bd-*) | Project-scoped dirs |
| **Machine-readable** | Fully structured | Partially (frontmatter only) |
| **Human-readable** | CLI output | Native markdown |

**Verdict**: Beads is superior for machine orchestration. Ledgers are superior for human review. Gastown can programmatically find "ready work"; SAI requires reading prose.

**SAI Gap**: We have no equivalent to `bd ready` (show actionable items) or `bd close --continue` (auto-advance workflows).

### 2.2 Workflow System

| Aspect | Gastown (MEOW) | SAI |
|--------|----------------|-----|
| **Definition** | TOML formulas | Skills + TodoWrite |
| **Instantiation** | Cook -> Pour/Wisp | Load skill, write todos |
| **Persistence** | Molecules (Git), Wisps (ephemeral) | Todos (session-only) |
| **Dependencies** | Explicit in formula | Implicit in todo order |
| **Loops** | Built-in (`loop-or-exit` step) | Ralph loops (separate) |
| **Gates** | Step entry/exit criteria | Completion gates (hooks) |
| **Templates** | Protomolecules | None |

**Verdict**: Gastown workflows are durable and composable. SAI workflows are ephemeral and ad-hoc. A formula in Gastown survives crashes; a TodoWrite list in SAI dies with the session.

**SAI Gap**: No workflow templates. No way to define "do X, then Y, then Z" that survives context resets.

### 2.3 Worker Model

| Role | Gastown | SAI Equivalent |
|------|---------|----------------|
| **Mayor** | Central concierge | Main CC session |
| **Polecats** | Ephemeral workers (worktrees) | Task subagents |
| **Crew** | Persistent workers (clones) | Zellij panes (manual) |
| **Witness** | Worker monitor | None (Ralph is self-check only) |
| **Refinery** | Merge queue | None |
| **Deacon** | Daemon/heartbeats | automation/manager.py (partial) |
| **Dogs** | Deacon helpers | None |

**Verdict**: Gastown has specialized supervision. SAI has general-purpose agents. Gastown workers are monitored externally; SAI agents self-monitor (less reliable).

**SAI Gaps**:
- No Witness equivalent (who unsticks stuck subagents?)
- No Refinery equivalent (merge queue is manual)
- No heartbeat system for worker health

### 2.4 Session Continuity

| Aspect | Gastown | SAI |
|--------|---------|-----|
| **Core principle** | GUPP: "Work on hook = run it" | Ledger: "Load state on start" |
| **Identity** | Agent Bead (persistent) | CC_WRAPPER_ID (per-wrapper) |
| **Handoff** | `gt handoff` (self-restart) | `/reload` skill |
| **Mail** | Structured messages (POLECAT_DONE, etc.) | None (stdout only) |
| **Resume** | `gt seance` (talk to predecessors) | Ledger review |

**Verdict**: GUPP is more proactive. If work exists, agent runs it without prompting. SAI waits for user input to check ledger.

**SAI Gap**: No equivalent to GUPP. SAI agents don't check for pending work on startup. No structured mail between sessions/agents.

---

## 3. What Gastown Solved That SAI Hasn't

### 3.1 The Merge Queue Problem

**Problem**: When swarming workers, they conflict on rebasing/merging. Final workers see unrecognizable baseline.

**Gastown Solution**: Refinery role. Dedicated engineer agent handles all merges, one at a time, to main. Test -> Rebase -> Merge -> Notify. No workers push directly.

**SAI Status**: Manual. We commit directly from main session or subagents. Conflicts resolved ad-hoc. No queue.

**Impact**: SAI can't effectively swarm 10+ workers on same repo.

### 3.2 Worker Supervision (Witness Pattern)

**Problem**: Workers get stuck. Context fills, they need restart. They wait politely for input that never comes.

**Gastown Solution**: Witness patrol. Runs in loop:
1. Check polecat health
2. Nudge stuck workers
3. Verify clean git state before kill
4. Report to Deacon

**SAI Status**: Ralph loops are self-checks (agent checks itself). No external observer. Stuck agents stay stuck until user notices.

**Impact**: SAI sessions can hang indefinitely. No automatic unsticking.

### 3.3 Durable Workflows

**Problem**: Agent crashes mid-workflow. Progress lost.

**Gastown Solution**: Molecules. Each step is a Bead. Agent restarts, finds position in molecule, continues.

```toml
# Example: 8-step polecat workflow
[[steps]]
id = "implement"
# Exit criteria: code compiles, no lint errors

[[steps]]
id = "self-review"
# Exit criteria: diff reviewed, no obvious bugs
```

**SAI Status**: TodoWrite lives in session. On crash, todos are gone. Must re-read ledger prose to understand state.

**Impact**: SAI can't run overnight autonomously. Crashes require human intervention to reconstruct state.

### 3.4 Ephemeral Orchestration (Wisps)

**Problem**: Patrol loops create noise. Every cycle creates audit trail. Git fills with orchestration garbage.

**Gastown Solution**: Wisps. Ephemeral beads stored in `.beads-wisp/` (never synced). Burn after completion. Optional squash to digest.

**SAI Status**: Everything goes to ledger or memory. Hook activity not separated from work activity.

**Impact**: SAI state files grow unbounded. No distinction between "operational noise" and "work product".

### 3.5 Propulsion Without Polling

**Problem**: Claude Code is polite. Waits for user input even when work is pending.

**Gastown Solution**: GUPP (Gastown Universal Propulsion Principle). Agents are prompted: "If work on hook, YOU RUN IT. No waiting." Plus nudge system if they don't.

**SAI Status**: Similar prompting in CLAUDE.md ("work until done or blocked"). But no external nudge system. No hook check on startup.

**Impact**: SAI agents sometimes sit idle when work is pending. Need user to say "continue".

### 3.6 Structured Inter-Agent Communication

**Problem**: Agents need to coordinate. "I'm done, merge my branch." "Your tests failed, fix it."

**Gastown Solution**: Mail protocol with typed messages:
- POLECAT_DONE -> Witness
- MERGE_READY -> Refinery
- MERGED -> Witness (triggers cleanup)
- HELP -> Escalation chain

**SAI Status**: No mail. Agents write to ledger. Other agents must poll ledger to see updates. No push notifications.

**Impact**: SAI coordination is pull-based (slow). Gastown is push-based (fast).

---

## 4. What SAI Has That Gastown Doesn't

### 4.1 Multi-Model Support

**SAI**: Routes to Claude (Opus/Haiku), Gemini, Codex, ZAI based on task type and usage.

**Gastown**: Claude-only. Single model, single provider.

**Advantage**: SAI can use cheaper/faster models for research, reserve expensive models for implementation.

### 4.2 Hook Extensibility

**SAI**: 7 event types (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop, SessionEnd, SubagentStop) with matcher patterns.

**Gastown**: Built-in patrol loops. Less extensible.

**Advantage**: SAI can intercept any lifecycle event and inject behavior.

### 4.3 Usage-Aware Routing

**SAI**: Tracks context usage. At 50%+, routes to CLI workers instead of subagents.

**Gastown**: No usage awareness mentioned. Hand off when full.

**Advantage**: SAI can maximize session lifetime by smart delegation.

### 4.4 Persistent Memory (mem0)

**SAI**: Vector + keyword search across sessions. Agents recall past discoveries.

**Gastown**: Git history only. No semantic search.

**Advantage**: SAI learns across sessions. "API rate limit is 100/min" stored once, recalled forever.

### 4.5 Self-Improvement Loop

**SAI**: Watcher job analyzes sessions, generates improvement suggestions with implementation plans.

**Gastown**: No equivalent mentioned.

**Advantage**: SAI infrastructure evolves automatically. Watcher detects patterns, suggests skills/rules.

### 4.6 Skill System (159+)

**SAI**: Rich library of specialized execution contexts. `/commit`, `/reload`, `/browser-automation`, etc.

**Gastown**: Custom agents only. No skill marketplace concept (though "Mol Mall" planned).

**Advantage**: SAI has ready-made solutions for common patterns.

---

## 5. Adoption Recommendations

### 5.1 High Priority (Gastown solutions to adopt)

#### A. GUPP-Style Propulsion

**What**: Check for pending work on session start. If work exists, run it without waiting.

**Implementation**:
```typescript
// hooks/session-start.ts addition
const pendingWork = await checkPendingWork();
if (pendingWork) {
  return {
    result: 'continue',
    message: `[GUPP] Pending work found: ${pendingWork.title}. Executing immediately.`
  };
}
```

**Effort**: Low (modify existing hook)

#### B. Durable Workflows (Molecule Analog)

**What**: Define workflows as structured TOML/JSON that survive crashes.

**Implementation**:
```toml
# ~/.claude/workflows/release.toml
name = "release"
steps = [
  { id = "version", criteria = "package.json version bumped" },
  { id = "changelog", criteria = "CHANGELOG.md updated" },
  { id = "test", criteria = "all tests pass" },
  { id = "build", criteria = "build succeeds" },
  { id = "publish", criteria = "npm publish succeeds" }
]
```

On resume, check last completed step and continue.

**Effort**: Medium (new system, but can build on TodoWrite)

#### C. Witness Pattern (External Worker Monitor)

**What**: Background job that checks on subagents, nudges stuck ones.

**Implementation**: Extend `automation/jobs/sai-watcher.ts` to:
1. Track active subagent sessions
2. Detect stuck patterns (no output for N minutes)
3. Send nudge via tmux/zellij
4. Escalate if still stuck

**Effort**: Medium (new automation job)

### 5.2 Medium Priority

#### D. Merge Queue (Refinery Analog)

**What**: Dedicated agent/job for merge operations.

**Implementation**:
- Create `automation/jobs/merge-queue.ts`
- Workers submit branches to queue
- Merge job processes sequentially: test -> rebase -> merge -> notify

**Effort**: High (complex coordination)

#### E. Structured Mail Protocol

**What**: Typed messages between agents instead of prose.

**Implementation**:
```typescript
// ~/.claude/lib/mail.ts
interface Message {
  type: 'TASK_DONE' | 'HELP' | 'HANDOFF' | 'MERGED';
  from: string;
  to: string;
  payload: any;
}

await sendMail({ type: 'TASK_DONE', to: 'main', payload: { result: 'success' } });
const messages = await checkMail('main');
```

**Effort**: Medium (new system)

#### F. Wisps (Ephemeral State)

**What**: Separate operational noise from work products.

**Implementation**:
- `~/.claude/state/ephemeral/` - Not synced, auto-cleaned
- `~/.claude/state/persistent/` - Synced, preserved
- Patrol hooks write to ephemeral only

**Effort**: Low (directory convention + hook changes)

---

## 6. Conclusion

Gastown and SAI solve the same fundamental problem: orchestrating multiple AI coding agents. Gastown is more cohesive with elegant primitives (Beads, MEOW, GUPP). SAI is more flexible with better extensibility (hooks, multi-model, skills).

**Key insight**: Gastown's GUPP and Molecules solve the "agent politeness problem" - agents that wait when they should work. SAI's Ralph loops are a workaround, not a solution.

**Recommended priority**:
1. GUPP-style propulsion (immediate impact)
2. Durable workflows (solve crash-resume)
3. Witness pattern (solve stuck agents)

The goal isn't to clone Gastown but to adopt its proven patterns where SAI is weaker, while preserving SAI's multi-model flexibility and hook extensibility.

---

## 7. Fork Strategy Analysis

### 7.1 Fork Feasibility Assessment

| Aspect | Finding | Difficulty | Decision |
|--------|---------|------------|----------|
| **License** | MIT - fully permissive | Easy | ✅ Fork allowed |
| **Language** | Go 1.24 | N/A | ✅ Keep Go (don't rewrite) |
| **Terminal** | Deeply coupled to tmux | Medium | ⚠️ Keep tmux initially, abstract later |
| **Claude coupling** | CLI spawn via tmux | Low | ✅ This is the magic - keep it |
| **State (Beads)** | JSONL + SQLite, Git-synced | N/A | ✅ Abandon ledgers, use Beads |
| **Modularity** | 37 internal packages | N/A | ✅ Don't modify core packages |

### 7.2 tmux vs Zellij Feasibility

**Research findings on swapping tmux → zellij:**

| Feature | tmux | Zellij | Gap |
|---------|------|--------|-----|
| Session management | Native | Native | ✅ Equivalent |
| Pane-targeted commands | `send-keys -t pane` | Focus + write-chars | ⚠️ **Serializes parallelism** |
| Scrollback capture | `capture-pane -p` | `dump-screen` + file | ⚠️ Extra step |
| Pane crash hooks | `set-hook pane-died` | WASM plugin required | ❌ **High effort** |
| Environment vars | `set-environment` | Plugin API | ⚠️ Plugin needed |
| Status bar | Native format strings | zjstatus plugin | ✅ Equivalent |
| Respawn pane | `respawn-pane` | Plugin `ReRunCommandInPane` | ⚠️ Plugin needed |

**Effort estimates:**
- Minimal migration (accepting limitations): 3-4 days
- Full migration with Zellij plugins: 5-7 days
- Abstraction layer (support both): +2-3 days

**Critical issue:** Gastown's parallel polecat model relies on `send-keys -t pane_id` to talk to multiple workers simultaneously. Zellij requires focusing a pane before writing, which **serializes operations**.

### 7.3 Recommended Approach: tmux + Zellij Coexistence

**Don't swap. Run both.**

```
Human (you)
    │
    ├── Zellij (human-in-the-loop work)
    │   └── Main CC session with full SAI hooks/skills
    │
    └── Gastown/tmux (autonomous swarm)
        ├── Polecat 1 → CC + SAI
        ├── Polecat 2 → CC + SAI
        └── Polecat N → CC + SAI
```

**Benefits:**
- Gastown's tmux integration stays untouched (no fork modifications)
- You keep Zellij for interactive work
- Each CC instance (tmux or Zellij) runs SAI enhancements
- Clear separation: tmux = autonomous factory, Zellij = workshop

---

## 8. Comprehensive Feature Matrix

### What to Keep from Each System

| Feature | Gastown | SAI | Keep From | Rationale |
|---------|---------|-----|-----------|-----------|
| **State storage** | Beads (JSONL/Git) | Ledgers (Markdown) | **Gastown** | Machine-queryable, Git-synced |
| **Workflows** | Molecules (TOML) | TodoWrite | **Gastown** | Durable, crash-survives |
| **Worker supervision** | Witness patrol | Ralph (self-check) | **Gastown** | External observer more reliable |
| **Merge coordination** | Refinery | Manual | **Gastown** | Essential for swarm |
| **Propulsion** | GUPP | Prompts only | **Gastown** | Eliminates idle agents |
| **Inter-agent comms** | Mail protocol | None | **Gastown** | Push > pull |
| **Ephemeral state** | Wisps | None | **Gastown** | Separates noise from work |
| **Multi-model routing** | None (Claude-only) | Gemini/Codex/ZAI | **SAI** | Cost optimization, speed |
| **Hook extensibility** | Built-in patrols | 7 event types | **SAI** | More flexible |
| **Usage-aware routing** | None | 70% threshold | **SAI** | Context preservation |
| **Persistent memory** | Git only | mem0 vector DB | **SAI** | Semantic recall |
| **Self-improvement** | None | Watcher | **SAI** | Infrastructure evolution |
| **Skill system** | Mol Mall (planned) | 159+ skills | **SAI** | Ready-made solutions |

### Features to Port: SAI → Gastown Fork

| Priority | SAI Feature | Integration Point | Effort | Value |
|----------|-------------|-------------------|--------|-------|
| **1** | Multi-model routing | Polecat spawn config | Medium | HIGH - cost savings |
| **1** | Persistent memory (mem0) | Crew workers | Medium-High | HIGH - learning |
| **2** | Usage-aware routing | Worker handoff logic | Low | Medium |
| **3** | Selective hooks | CC settings per worker | High | Medium |
| **4** | Watcher (self-improve) | As Gastown plugin | Medium | Medium-High |
| **5** | Key skills | As Gastown formulas | Low | Medium |

### Features NOT to Port (Gastown does it better)

| SAI Feature | Why NOT Port | Gastown Equivalent |
|-------------|--------------|-------------------|
| Ledgers | Less queryable | Use Beads |
| TodoWrite | Session-bound | Use Molecules |
| Ralph loops | Self-check unreliable | Use Witness patrol |
| Hook-based GUPP | Partial solution | Use native GUPP |
| Manual merges | Doesn't scale | Use Refinery |

---

## 9. Integration Architecture (Fork Plan)

### Phase 1: Fork + CC Enhancement (Week 1-2)

```
1. Fork steveyegge/gastown
2. Keep all Go code untouched
3. Replace .claude/ with SAI configuration:
   - Copy hooks/ (all 7 event types)
   - Copy skills/ (start with essential 20)
   - Copy agents/ (Task subagent definitions)
   - Copy rules/ (behavioral constraints)
   - Merge AGENTS.md (Gastown + SAI instructions)
4. Test: Single polecat with SAI enhancements
```

### Phase 2: Multi-Model Integration (Week 3-4)

```
1. Add model routing to polecat spawn:
   - Research tasks → gemini CLI
   - Implementation → claude (default)
   - Planning → codex
2. Configure via Beads metadata:
   bd new --model gemini "Research API endpoints"
3. Modify internal/polecat/manager.go (minimal):
   - Read model from bead
   - Spawn appropriate CLI
```

### Phase 3: Memory + Watcher (Week 5-8)

```
1. Integrate mem0 for Crew workers:
   - Persistent workers contribute memories
   - Polecats read but don't write (ephemeral)
2. Add Watcher as Gastown job:
   - Analyze completed molecules
   - Suggest formula improvements
   - Auto-create new formulas from patterns
```

---

## 10. Open Questions

1. **Upstream contribution**: Should improvements be PRed back to Gastown?
2. **Tool-use conventions**: Different models have different tool calling - how to normalize?
3. **Memory scope**: Should polecats contribute to mem0 or only Crew?
4. **Skill→Formula mapping**: How to convert SAI skills to Gastown formulas?
5. **Zellij integration**: Worth adding as optional backend, or keep tmux-only?

---

## 11. Conclusion

**Recommendation: Fork Gastown, enhance with SAI**

The systems are complementary:
- **Gastown** solves orchestration (outside CC): supervision, merges, workflows, state
- **SAI** solves enhancement (inside CC): hooks, skills, memory, multi-model

**Don't rewrite. Don't swap tmux. Merge CC configurations.**

Each Gastown-spawned CC instance becomes a SAI-enhanced instance. You get:
- Gastown's proven orchestration primitives
- SAI's multi-model routing and persistent memory
- Best of both worlds

**Next step**: Fork repo, test SAI hooks in a single polecat, iterate. 