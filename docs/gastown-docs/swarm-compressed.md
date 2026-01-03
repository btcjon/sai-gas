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

See: [Convoys](convoy.md), [Propulsion Principle](propulsion-principle.md)
