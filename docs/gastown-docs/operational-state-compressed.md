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
