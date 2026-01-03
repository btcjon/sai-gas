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
