# SAI Capabilities to Port to Gas Town

> Extracted from sai-vs-gastown.md analysis
> Last updated: 2026-01-03

## Overview

SAI operates at the **Claude Code layer** (hooks, skills, memory) while Gas Town operates at the **orchestration layer** (tmux, workers, supervision). These capabilities can enhance each CC instance spawned by Gas Town.

### Status Key

| Status | Meaning |
|--------|---------|
| **NATIVE** | Already works via global `~/.claude/` scope |
| **ENABLED** | Works but requires `GASTOWN_ENABLE_*` env var |
| **TODO** | Needs implementation work |

---

## Quick Status Summary

| Capability | Status | Enable With |
|------------|--------|-------------|
| Hook system (7 events) | NATIVE | (auto, but most skip in sai-gas) |
| Skill system (159+) | NATIVE | `Skill("name")` |
| Specialized agents (18) | NATIVE | `Task(subagent_type="...")` |
| mem0 memory | NATIVE | (auto) |
| CLI workers (gemini/codex/zcc) | NATIVE | Just use commands |
| Usage-aware routing | ENABLED | `GASTOWN_ENABLE_USAGE=1` |
| Multi-model routing | ENABLED | See `docs/multi-model-routing.md` |
| Watcher (self-improve) | TODO | Port as Deacon dog |
| Artifact indexing | SUPERSEDED | Use Beads instead |

---

## Priority 1: High Value, Clear Integration Path

### 1. Multi-Model Routing ✅ ENABLED

**Status**: Available via CLI workers. See `docs/multi-model-routing.md` for full guide.

**What it does**: Routes tasks to different AI models based on task type and usage limits.

| Task Type | Model | Command |
|-----------|-------|---------|
| Research | Gemini Flash | `gemini -y "query" 2>/dev/null` |
| Implementation | Claude Opus | native CC |
| Planning | Codex | `codex gpt-5.2-codex "..."` |
| Simple tasks | Haiku | `zcc -p "task"` |

**Value**: HIGH - Gemini is ~10x cheaper than Opus

---

### 2. Persistent Memory (mem0) ✅ NATIVE

**Status**: Works automatically via global `~/.claude/` hooks.

**What it does**: Vector + keyword search across sessions. Agents recall past discoveries.

**Examples**:
- "API rate limit is 100/min" - stored once, recalled forever
- "User prefers tabs over spaces" - remembered
- "Project uses pnpm not npm" - won't make same mistake twice

**Enable explicitly**: `GASTOWN_ENABLE_MEMORY=1`

**Value**: HIGH - eliminates repetitive re-learning

---

### 3. Hook System (7 Event Types) ✅ NATIVE (selective)

**Status**: Global hooks exist but most skip in sai-gas via `isGastownMode()` check.

**What it does**: Intercepts CC lifecycle events and injects custom behavior.

| Event | Use Case | Status in sai-gas |
|-------|----------|-------------------|
| `SessionStart` | Load context, check pending work | Runs (gt prime) |
| `UserPromptSubmit` | Pre-process user input, load skills | Runs (mail check) |
| `PreToolUse` | Block dangerous operations | Skipped |
| `PostToolUse` | Auto-index artifacts, trigger side-effects | Skipped |
| `Stop` | Verify completion, run gates | Skipped (Ralph) |
| `SubagentStop` | Capture subagent results | Skipped |
| `SessionEnd` | Cleanup, sync state | Skipped |

**Enable specific hooks**: Use `GASTOWN_ENABLE_*` env vars

**Value**: HIGH - enables all other capabilities

---

## Priority 2: Medium Value, Moderate Effort

### 4. Usage-Aware Routing ✅ ENABLED

**Status**: Enable with `GASTOWN_ENABLE_USAGE=1`

**What it does**: Tracks context usage. At 50%+ capacity, routes to CLI workers instead of subagents to preserve context.

```
Context 0-25%:  Use Opus subagents freely
Context 25-50%: Warning, start delegating more
Context 50-70%: Use CLI workers (gemini, codex)
Context 70%+:   Auto-reload, continue from ledger
```

**State file**: `~/.claude/state/session-tool-usage.json`

**Value**: Medium - extends session lifetime

---

### 5. Skill System (159+ Skills) ✅ NATIVE

**Status**: Works automatically via global `~/.claude/skills/`

**What it does**: Specialized execution contexts loaded on-demand.

**Usage**: `Skill("skill-name")` or `/skill-name`

**Essential skills available**:

| Skill | Purpose |
|-------|---------|
| `commit` | Git commits without Claude attribution |
| `browser-automation` | CDP/Playwright web interaction |
| `research` | Web research with source citation |
| `ai-image-generation` | Generate images via kie.ai |
| `code-search` | ast-grep + text search |
| `cloudflare-services` | Deploy to Cloudflare |
| `aws-services` | AWS operations |

**Value**: Medium - ready-made solutions

---

### 6. Watcher (Self-Improvement) ⏳ TODO

**Status**: Needs porting as Deacon dog

**What it does**: Background job analyzes sessions, detects patterns, suggests improvements.

**Outputs**:
- New skill suggestions with implementation plans
- Rule violations detected
- Performance bottlenecks identified
- Common patterns that should become skills

**Integration point**: As Deacon dog or scheduled job

**Value**: Medium-High - infrastructure evolves automatically

---

### 7. Agent Memory Context

**What it does**: Agents automatically recall relevant memories from past sessions.

```typescript
// On session start, query mem0 for relevant context
const memories = await mem0.search(currentTask);
// Inject into system prompt
```

**Integration point**: SessionStart hook

**Effort**: Medium
**Value**: Medium - reduces re-discovery

---

## Priority 3: Nice to Have

### 8. Specialized Task Agents (18 Types) ✅ NATIVE

**Status**: Works via `Task(subagent_type="...")` tool

**What it does**: Pre-configured agents for specific domains.

| Agent Type | Specialty |
|------------|-----------|
| `frontend-developer` | React/Vue/Angular, UI/UX |
| `backend-architect` | API design, databases |
| `seo-optimizer` | Search optimization |
| `apex-copywriter` | Marketing copy |
| `research-agent` | Deep research |
| `docs-manager` | Documentation |

**Usage**: `Task(subagent_type="frontend-developer", prompt="...")`

**Value**: Medium - domain expertise out of the box

---

### 9. Ralph Loops (Iterative Verification) ⏸️ SKIPPED

**Status**: Disabled in sai-gas - Gas Town's Witness is superior

**What it does**: Runs N iterations checking completion gates before allowing stop.

**Gates checked**:
- Tests pass
- No errors in output
- Todos completed
- Verification evidence present

**Recommendation**: Use Gas Town's Witness pattern instead (external verification > self-check)

**Value**: Low-Medium (Witness is better)

---

### 10. Artifact Indexing ⏸️ SUPERSEDED

**Status**: Beads handles this better

**What it does**: PostToolUse hook indexes created files for full-text search.

**Why Beads is better**:
- Already queryable via `bd list`, `bd show`, SQLite
- Already git-synced
- No separate index to maintain
- Native search: `sqlite3 .beads/beads.db "SELECT * FROM issues WHERE title LIKE '%query%'"`

**Recommendation**: Use Beads for all artifact tracking, not SAI indexing

---

### 11. CLI Workers (gemini, codex, zcc) ✅ NATIVE

**Status**: Already available in PATH

**What it does**: Command-line wrappers for external AI models.

```bash
gemini -y "research API endpoints" 2>/dev/null
codex gpt-5.2-codex "plan implementation"
zcc -p "quick task"
```

**Paths**:
- `gemini` → `/home/dev/.npm-global/bin/gemini`
- `codex` → `/home/dev/.npm-global/bin/codex`
- `zcc` → `/home/dev/.claude/scripts/zcc`

**Value**: Medium - enables multi-model

---

## Not Recommended to Port

These SAI features are superseded by Gas Town equivalents:

| SAI Feature | Gas Town Equivalent | Reason |
|-------------|---------------------|--------|
| Ledgers | Beads | Beads are queryable, Git-synced |
| TodoWrite | Molecules | Molecules survive crashes |
| Hook-based GUPP | Native GUPP | Gas Town's is more complete |
| Manual merges | Refinery | Refinery handles conflicts |

---

## Implementation Roadmap

### Phase 1: Core Integration (Week 1)
1. Copy hook system to `.claude/hooks/`
2. Copy essential skills to `.claude/skills/`
3. Merge AGENTS.md with SAI prompting
4. Test single polecat with enhancements

### Phase 2: Multi-Model (Week 2-3)
1. Add model routing to polecat config
2. Configure bead metadata for model selection
3. Test gemini/codex for research tasks

### Phase 3: Memory (Week 4-6)
1. Integrate mem0 for Crew workers
2. Configure polecats as read-only
3. Add memory recall to SessionStart

### Phase 4: Watcher (Week 7-8)
1. Port watcher as Deacon dog
2. Configure session analysis
3. Auto-generate formula suggestions

---

## Quick Reference: Files to Copy

```
# From SAI to sai-gas
~/.claude/hooks/              → .claude/hooks/
~/.claude/skills/             → .claude/skills/ (subset)
~/.claude/agents/             → .claude/agents/
~/.claude/rules/              → .claude/rules/
~/.claude/lib/                → .claude/lib/ (for mem0)
~/.claude/settings.json       → .claude/settings.json (merge)
```

---

## Success Metrics

| Metric | Current (Gas Town) | Target (SAI-enhanced) |
|--------|-------------------|----------------------|
| Cost per task | $X (Opus only) | $X/3 (multi-model) |
| Re-learning rate | High | Low (memory) |
| Stuck agent recovery | Manual | Auto (hooks + witness) |
| Pattern→skill time | Manual | Auto (watcher) |