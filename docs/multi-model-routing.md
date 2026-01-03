# Multi-Model Routing for Gas Town

## Enabling SAI Features

Set these env vars to enable SAI capabilities in sai-gas:

```bash
# In your shell config (~/.bashrc or ~/.zshrc)
export GASTOWN_ENABLE_USAGE=1      # Track tool usage for context-aware routing
export GASTOWN_ENABLE_MEMORY=1     # Enable mem0 persistent memory
export GASTOWN_ENABLE_MULTIMODEL=1 # Enable multi-model routing logic
export GASTOWN_ENABLE_SKILLS=1     # Enable skill injection
```

Or set per-session:
```bash
GASTOWN_ENABLE_USAGE=1 claude
```

---

## Available Models

| CLI | Model | Best For | Speed | Cost |
|-----|-------|----------|-------|------|
| `gemini` | gemini-3-flash-preview | Research, exploration | Fast | Cheap |
| `codex` | gpt-5.2-codex | Planning, reasoning | Medium | Medium |
| `zcc` | claude-3-haiku | Quick tasks | Fast | Very cheap |
| (native) | claude-opus | Implementation | Slow | Expensive |

## Routing Decision Matrix

```
Task Type           → Use This
─────────────────────────────────
Research/Search     → gemini -y "query"
File exploration    → gemini -y "find X in codebase"
Planning/Design     → codex "plan the implementation"
Quick/trivial       → zcc -p "simple task"
Implementation      → native CC (Opus)
Code editing        → native CC (Opus)
```

## Usage Patterns

### Research (delegate to Gemini)
```bash
# Non-interactive, auto-approve
gemini -y "what are the best practices for Go error handling" 2>/dev/null
```

### Parallel Research
```bash
gemini -y "research API authentication patterns" 2>/dev/null &
gemini -y "find examples of rate limiting in Go" 2>/dev/null &
wait
```

### Planning (delegate to Codex)
```bash
codex gpt-5.2-codex "plan how to implement convoy auto-close feature"
```

### Quick Tasks (delegate to Haiku)
```bash
zcc -p "summarize this error message: <error>"
```

## When to Delegate

| Context Used | Strategy |
|--------------|----------|
| 0-25% | Use Opus freely |
| 25-50% | Start delegating research |
| 50-70% | Delegate all non-implementation |
| 70%+ | Delegate everything, prepare for reload |

## Integration with Polecats

Workers can specify preferred model in bead metadata:

```bash
# Create research task for Gemini worker
bd create --title="Research API patterns" --type=task

# In polecat startup, check task type and route
if [[ "$TASK_TYPE" == "research" ]]; then
  gemini -y "$TASK_DESCRIPTION" 2>/dev/null
else
  # Normal CC execution
fi
```

## Cost Comparison (Approximate)

| Model | Cost/1M tokens | Relative |
|-------|----------------|----------|
| Opus | $15 | 1x |
| Haiku | $0.25 | 60x cheaper |
| Gemini Flash | $0.10 | 150x cheaper |
| Codex | $2 | 7.5x cheaper |

**Example savings**:
- 10 research queries via Gemini instead of Opus = ~$1.50 saved
- 100 research queries = ~$15 saved (1 full Opus context worth)
