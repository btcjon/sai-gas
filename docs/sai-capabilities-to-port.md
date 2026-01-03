# SAI Capabilities in Gas Town

> Extracted from sai-vs-gastown.md analysis
> Last updated: 2026-01-03

## Overview

SAI operates at the **Claude Code layer** (hooks, skills, memory) while Gas Town operates at the **orchestration layer** (tmux, workers, supervision). These capabilities can enhance each CC instance spawned by Gas Town.

### How Gas Town Interacts with SAI Hooks

**Gas Town automatically injects `GASTOWN_MODE=1`** when spawning CC sessions (polecats, crew). This is set in `internal/config/loader.go:743`.

All SAI hooks call `skipIfGastown()` at startup, which checks for this env var and exits early if set. This means:

- **Mayor sessions** (direct CC in sai-gas): SAI hooks run normally
- **gt-spawned sessions** (polecats/crew): SAI hooks skip, vanilla CC behavior

### Status Key

| Status | Meaning |
|--------|---------|
| **NATIVE** | Works via global `~/.claude/` scope, no hook needed |
| **SKIPPED** | Hook exists but self-skips when GASTOWN_MODE=1 |
| **GT-NATIVE** | Handled by Gas Town, not SAI |

---

## Quick Status Summary

| Capability | Status | Notes |
|------------|--------|-------|
| Skill system (161+) | NATIVE | `~/.claude/skills/` always available |
| CLI workers (gemini/codex/zcc) | NATIVE | In PATH, always available |
| Specialized agents (20+) | NATIVE | `Task(subagent_type="...")` works |
| mem0 memory hooks | SKIPPED | skipIfGastown() in all memory hooks |
| Usage-aware routing | SKIPPED | skipIfGastown() unless GASTOWN_ENABLE_USAGE=1 |
| Continuity hooks | SKIPPED | Beads replaces this in gt context |
| Ralph loops | SKIPPED | GT Witness replaces this |
| Artifact indexing | GT-NATIVE | Beads handles this |

---

## Always Available (NATIVE)

These work in any CC session because they don't depend on hooks:

### 1. Skill System (161+ Skills)

**Works via**: Global `~/.claude/skills/` directory

**Usage**: `Skill("skill-name")` or `/skill-name`

| Skill | Purpose |
|-------|---------|
| `commit` | Git commits without Claude attribution |
| `browser-automation` | CDP/Playwright web interaction |
| `research` | Web research with source citation |
| `ai-image-generation` | Generate images via kie.ai |
| `code-search` | ast-grep + text search |
| `cloudflare-services` | Deploy to Cloudflare |
| `aws-services` | AWS operations |

---

### 2. CLI Workers

**Works via**: Executables in PATH

```bash
gemini -y "research API endpoints" 2>/dev/null  # Gemini Flash
codex gpt-5.2-codex "plan implementation"        # GPT-5.2 Codex
zcc -p "quick task"                              # Claude Haiku via zai
```

**Paths**:
- `gemini` → `/home/dev/.npm-global/bin/gemini`
- `codex` → `/home/dev/.npm-global/bin/codex`
- `zcc` → `/home/dev/.claude/scripts/zcc`

---

### 3. Specialized Task Agents (20+ Types)

**Works via**: `Task(subagent_type="...")` tool

| Agent Type | Specialty |
|------------|-----------|
| `frontend-developer` | React/Vue/Angular, UI/UX |
| `backend-architect` | API design, databases |
| `seo-optimizer` | Search optimization |
| `apex-copywriter` | Marketing copy |
| `research-agent` | Deep research |
| `docs-manager` | Documentation |
| `gmail-manager` | Email operations |
| `google-ads-manager` | PPC optimization |
| `landing-page-architect` | Conversion design |
| `nbp-imagen` | AI image generation |

Full list in `~/.claude/agents/`

---

## Skipped in Gas Town (SKIPPED)

These hooks exist but self-skip when gt spawns sessions:

### 4. Memory System (mem0)

**Hooks affected**: `memory-context-loader.ts`, `memory-posttool-capture.ts`, `session-memory-capture.ts`

**Why skipped**: Polecats are ephemeral. Memory is valuable for long-lived workers, not single-task polecats.

**Could enable with**: `GASTOWN_ENABLE_MEMORY=1` (mechanism exists, not wired up)

---

### 5. Usage-Aware Routing

**Hook affected**: `cumulative-usage-tracker.ts`, `usage-aware-router.ts`

**What it does**: Tracks context usage, routes to cheaper models at high usage.

**Enable with**: `GASTOWN_ENABLE_USAGE=1` (this one IS wired up)

---

### 6. Continuity Hooks

**Hooks affected**: `continuity-session-start.ts`, `continuity-session-end.ts`, `continuity-precompact.ts`

**Why skipped**: Gas Town uses Beads for continuity, not SAI ledgers.

---

### 7. Ralph Loops

**Hooks affected**: `auto-ralph-init.ts`, `enhanced-ralph-stop.ts`

**Why skipped**: Gas Town's Witness provides superior external verification.

---

## Handled by Gas Town (GT-NATIVE)

These are replaced by Gas Town systems:

| SAI Feature | Gas Town Equivalent | Why GT is Better |
|-------------|---------------------|------------------|
| Ledgers | Beads | Queryable, Git-synced, SQLite |
| TodoWrite | Molecules | Survive crashes, multi-agent |
| Ralph self-check | Witness | External verification |
| Manual merges | Refinery | Automated conflict resolution |

---

## Hook Compatibility Reference

### Would Work (if skipIfGastown removed)

| Hook | Purpose | Uses Beads |
|------|---------|------------|
| session-start.sh | Skills list, delegation reminder | N/A |
| continuity-session-start.ts | Show available beads | Yes |
| reload-resume.ts | Hot-reload context injection | Yes |
| pre-clear-gate.ts | Block /clear without bead bound | Yes |
| continuity-precompact.ts | Auto-reload at 70% | Yes |
| continuity-session-end.ts | Session summary to bead | Yes |
| skill-activation-prompt.ts | Skill loading hints | N/A |
| verification-enforcer.ts | Verify work before stop | N/A |
| session-memory-capture.ts | mem0 persistence | N/A |
| session-outcome.ts | Track outcomes | N/A |
| usage-aware-router.ts | Cost gating | N/A |
| subagent-stop-continuity.ts | Agent logging | N/A |
| cumulative-usage-tracker.ts | Usage tracking | N/A |
| agent-auto-suggest.ts | Helpful suggestions | N/A |
| post-tool-use-tracker.ts | Tool tracking | N/A |
| typescript-preflight.ts | TS error checks | N/A |
| watcher-snapshot.ts | Watcher state | N/A |

### Would NOT Work (SAI-specific systems)

| Hook | Why Incompatible |
|------|------------------|
| workflow-initializer.ts | AWID workflow - GT has molecules |
| workflow-progress.ts | AWID workflow - GT has molecules |
| auto-ralph-init.ts | Ralph loops - GT has polecats |
| enhanced-ralph-stop.ts | Ralph loops - GT has polecats |
| idle-detector.ts | Queue manager - GT has witness/deacon |
| self-improvement.ts | SAI handoffs dir structure |

---

## Project Settings Note

The project `.claude/settings.json` contains:

```json
{
  "env": {
    "DISABLE_WORKFLOW_HOOKS": "1",
    "RALPH_DISABLED": "1"
  }
}
```

**These are now redundant** since gt auto-injects `GASTOWN_MODE=1` which triggers `skipIfGastown()` in all hooks. The project settings were created before the gastown detection was fully implemented.

---

## Enabling SAI Features in Gas Town

The `~/.claude/hooks/lib/gastown-check.ts` supports feature flags:

```typescript
GASTOWN_ENABLE_MEMORY=1     // Enable mem0 even in Gastown mode
GASTOWN_ENABLE_MULTIMODEL=1 // Enable multi-model routing
GASTOWN_ENABLE_SKILLS=1     // Enable skill system (already native)
GASTOWN_ENABLE_USAGE=1      // Enable usage tracking
```

**Current implementation**: Only `GASTOWN_ENABLE_USAGE` is actually checked by any hook. The others exist in the API but no hooks use them yet.

To enable: Add to rig config or set in gt spawn environment.

---

## Success Metrics

| Metric | Current (vanilla GT) | With SAI features |
|--------|---------------------|-------------------|
| Cost per task | Opus only | Multi-model (~3x cheaper) |
| Re-learning rate | High | Low (memory) |
| Pattern→skill time | Manual | Auto (watcher) |

---

## Future Integration Options

1. **Enable mem0 for Crew** (persistent workers benefit from memory)
2. **Wire up GASTOWN_ENABLE_MEMORY** in memory hooks
3. **Port Watcher as Deacon dog** for self-improvement
4. **Remove redundant project settings.json overrides**
