# Hot-Reload vs Built-in Auto-Compact

Decision: **Hybrid approach** - Use built-in auto-compact with checkpoint detection, keep hot-reload for config changes.

## Architecture

| Concern | Solution |
|---------|----------|
| Context exhaustion | Built-in auto-compact + checkpoint system |
| Config/agent reload | Hot-reload (`/reload` skill) |

## How It Works

### Checkpoint System (for auto-compact)

```
Context at 72%+:
├─ UserPromptSubmit hook fires
├─ Capture checkpoint: bead, todos, files, conversation summary
└─ Save to ~/.claude/state/context-checkpoint.json

Built-in auto-compact fires (~80-90%):
├─ CC compresses conversation internally
└─ Context drops (e.g., 85% → 35%)

Next UserPromptSubmit:
├─ Hook detects drop: last_pct=85, current_pct=35
├─ COMPACTION DETECTED!
├─ Read checkpoint, inject as <system-reminder>
└─ Clear checkpoint (one-time use)
```

### Thresholds

| Threshold | Value | Purpose |
|-----------|-------|---------|
| `CHECKPOINT_THRESHOLD` | 72% | Start capturing checkpoints |
| `COMPACTION_DETECT_HIGH` | 70% | Was above this... |
| `COMPACTION_DETECT_LOW` | 50% | ...now below this = compaction |
| `AUTO_RELOAD_THRESHOLD` | 85% | Hot-reload for `/reload` skill |

### What Gets Captured

```typescript
checkpoint: {
  bead_id: string | null,     // Current bound bead
  cwd: string,                // Working directory
  context_pct: number,        // Context when captured
  user_task: string,          // From bead title/[TASK] comment
  todos: string[],            // In-progress + pending todos
  files_touched: string[],    // Recent Write/Edit tool calls
  conversation_summary: string // LLM-compressed summary
}
```

### Injection Format

After compaction, user sees:

```
Session was auto-compacted. Restoring context from checkpoint:

**Bead:** gt-jixim
**Task:** Implement checkpoint system for auto-compact
**Todos:** Update docs, Close bead
**Files touched:** continuity-precompact.ts, HOT-RELOAD-VS-AUTOCOMPACT.md

**Session summary:**
[LLM-compressed conversation focusing on decisions, progress, next steps]

Continue working on the task above.
```

## Hot-Reload (still available)

Hot-reload (`/reload` skill) remains for:
- Config changes (CLAUDE.md, settings)
- Agent updates (skill files)
- Manual context reset

Triggered via:
- `/reload` skill
- Ctrl+C → [r] restart in cc-wrapper

## Files

| File | Purpose |
|------|---------|
| `~/.claude/hooks/continuity-precompact.ts` | Checkpoint capture + detection |
| `~/.claude/state/context-checkpoint.json` | Checkpoint state |
| `~/.claude/scripts/cc-wrapper.sh` | Hot-reload wrapper (kept) |
| `~/.claude/scripts/cc-reload-supervisor.ts` | Hot-reload supervisor (kept) |

## Comparison

| Feature | Hot-Reload Only | Hybrid (Chosen) |
|---------|-----------------|-----------------|
| Bead continuity | Auto-restore on restart | Checkpoint injection |
| Context capture | Before reload | Before auto-compact |
| Auto-continue | Via wrapper prompt | Via system-reminder |
| Config reload | Yes | Yes (via /reload) |
| Infrastructure | Complex | Moderate |
| Seamless | Restart required | In-place |

## Why Hybrid?

1. **Built-in auto-compact is seamless** - No session restart, no lost state
2. **Our detection fills the gap** - Injects context CC doesn't know about
3. **Hot-reload still useful** - Config changes need full restart
4. **Best of both worlds** - Auto-compact for context, hot-reload for config
