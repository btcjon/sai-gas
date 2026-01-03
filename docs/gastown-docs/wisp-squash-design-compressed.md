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
- Formula-defined templates - extensibility, workflow-specific metrics
- Retain forever as rollups - capability ledger needs long-term history
