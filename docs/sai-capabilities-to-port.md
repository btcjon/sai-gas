# SAI Capabilities to Port to Gas Town

> Extracted from sai-vs-gastown.md analysis

## Overview

SAI operates at the **Claude Code layer** (hooks, skills, memory) while Gas Town operates at the **orchestration layer** (tmux, workers, supervision). These capabilities can enhance each CC instance spawned by Gas Town.

---

## Priority 1: High Value, Clear Integration Path

### 1. Multi-Model Routing

**What it does**: Routes tasks to different AI models based on task type and usage limits.

| Task Type | Model | Rationale |
|-----------|-------|-----------|
| Research | Gemini Flash | Fast, cheap, good at search |
| Implementation | Claude Opus | Best at coding |
| Planning | Codex | Deep reasoning |
| Simple tasks | Haiku | Fast, very cheap |

**Integration point**: Polecat spawn config or bead metadata
```bash
bd create --model=gemini "Research API endpoints"
```

**Effort**: Medium
**Value**: HIGH - significant cost savings (Gemini ~10x cheaper than Opus)

---

### 2. Persistent Memory (mem0)

**What it does**: Vector + keyword search across sessions. Agents recall past discoveries.

**Examples**:
- "API rate limit is 100/min" - stored once, recalled forever
- "User prefers tabs over spaces" - remembered
- "Project uses pnpm not npm" - won't make same mistake twice

**Integration point**: Crew workers contribute memories; polecats read-only

**Effort**: Medium-High
**Value**: HIGH - eliminates repetitive re-learning

---

### 3. Hook System (7 Event Types)

**What it does**: Intercepts CC lifecycle events and injects custom behavior.

| Event | Use Case |
|-------|----------|
| `SessionStart` | Load context, check pending work |
| `UserPromptSubmit` | Pre-process user input, load skills |
| `PreToolUse` | Block dangerous operations |
| `PostToolUse` | Auto-index artifacts, trigger side-effects |
| `Stop` | Verify completion, run gates |
| `SubagentStop` | Capture subagent results |
| `SessionEnd` | Cleanup, sync state |

**Integration point**: `.claude/settings.json` per worker type

**Effort**: Low (copy config)
**Value**: HIGH - enables all other capabilities

---

## Priority 2: Medium Value, Moderate Effort

### 4. Usage-Aware Routing

**What it does**: Tracks context usage. At 50%+ capacity, routes to CLI workers instead of subagents to preserve context.

```
Context 0-25%:  Use Opus subagents freely
Context 25-50%: Warning, start delegating more
Context 50-70%: Use CLI workers (gemini, codex)
Context 70%+:   Auto-reload, continue from ledger
```

**Integration point**: Worker handoff logic

**Effort**: Low
**Value**: Medium - extends session lifetime

---

### 5. Skill System (159+ Skills)

**What it does**: Specialized execution contexts loaded on-demand.

**Essential skills to port**:

| Skill | Purpose |
|-------|---------|
| `commit` | Git commits without Claude attribution |
| `browser-automation` | CDP/Playwright web interaction |
| `research` | Web research with source citation |
| `ai-image-generation` | Generate images via kie.ai |
| `code-search` | ast-grep + text search |
| `cloudflare-services` | Deploy to Cloudflare |
| `aws-services` | AWS operations |

**Integration point**: As Gas Town formulas or `.claude/skills/`

**Effort**: Low (copy files)
**Value**: Medium - ready-made solutions

---

### 6. Watcher (Self-Improvement)

**What it does**: Background job analyzes sessions, detects patterns, suggests improvements.

**Outputs**:
- New skill suggestions with implementation plans
- Rule violations detected
- Performance bottlenecks identified
- Common patterns that should become skills

**Integration point**: As Gas Town job or Deacon dog

**Effort**: Medium
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

### 8. Specialized Task Agents (18 Types)

**What it does**: Pre-configured agents for specific domains.

| Agent Type | Specialty |
|------------|-----------|
| `frontend-developer` | React/Vue/Angular, UI/UX |
| `backend-architect` | API design, databases |
| `seo-optimizer` | Search optimization |
| `apex-copywriter` | Marketing copy |
| `research-agent` | Deep research |
| `docs-manager` | Documentation |

**Integration point**: Polecat templates

**Effort**: Medium
**Value**: Medium - domain expertise out of the box

---

### 9. Ralph Loops (Iterative Verification)

**What it does**: Runs N iterations checking completion gates before allowing stop.

**Gates checked**:
- Tests pass
- No errors in output
- Todos completed
- Verification evidence present

**Note**: Gas Town's Witness pattern is superior for external verification. Ralph is useful as self-check fallback.

**Integration point**: Stop hook

**Effort**: Low
**Value**: Low-Medium (Witness is better)

---

### 10. Artifact Indexing

**What it does**: PostToolUse hook indexes created files for full-text search.

**Use case**: Find past work, search handoffs, locate decisions

**Integration point**: PostToolUse hook

**Effort**: Low
**Value**: Low-Medium

---

### 11. CLI Workers (gemini, codex, zcc)

**What it does**: Command-line wrappers for external AI models.

```bash
gemini -y "research API endpoints" 2>/dev/null
codex gpt-5.2-codex "plan implementation"
zcc -p "quick task"
```

**Integration point**: Already available, just document usage

**Effort**: None (already works)
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

*native by default = meaning these are core cc  abilities across all b/c they are in global/user scope right?

| Priority | Capability               | Effort      | Value       |
  |----------|--------------------------|-------------|-------------|
  | P1       | Multi-model routing      | Medium      | HIGH        |
  | P1       | Persistent memory (mem0) | Medium-High | HIGH        | *native by default
  | P1       | Hook system (7 events)   | Low         | HIGH        | *native by default (disabled by envars currently)
  | P2       | Usage-aware routing      | Low         | Medium      |
  | P2       | Skill system (159+)      | Low         | Medium      | *native by default
  | P2       | Watcher (self-improve)   | Medium      | Medium-High |
  | P2       | Agent memory context     | Medium      | Medium      |
  | P3       | Specialized agents (18)  | Medium      | Medium      | *native by default
  | P3       | Ralph loops              | Low         | Low-Medium  |
  | P3       | Artifact indexing        | Low         | Low-Medium  |