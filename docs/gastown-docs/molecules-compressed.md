# Molecules

Workflow templates for multi-step work.

## Lifecycle

```
Formula (TOML) ─── bd cook ───► Protomolecule (frozen)
                                    │
                    ├── bd mol pour ───► Molecule (persistent) ─► bd squash ─► Digest
                    └── bd mol wisp ───► Wisp (ephemeral) ──────► bd burn ─► (gone)
```

| Term | Description |
|------|-------------|
| Formula | Source TOML template |
| Protomolecule | Frozen, ready to instantiate |
| Molecule | Active workflow with trackable steps |
| Wisp | Ephemeral molecule (never synced) |
| Digest | Squashed summary of completed mol |

## Navigation

```bash
bd mol current              # Where am I?
bd mol current gt-abc       # Status of specific mol
```

Output:
```
  ✓ gt-abc.1: Design
  ✓ gt-abc.2: Scaffold
  → gt-abc.3: Implement [in_progress] <- YOU ARE HERE
  ○ gt-abc.4: Tests
```

## Transitions

```bash
bd close gt-abc.3 --continue   # Close and auto-advance to next
bd close gt-abc.3 --no-auto    # Close without advancing
```

## Commands

**Beads (data):**
```bash
bd formula list / show / cook
bd mol list / show / pour / wisp / bond / squash / burn / current
```

**Agent (execution):**
```bash
gt hook                  # What's on my hook
gt mol current           # What to work on next
gt mol attach / detach   # Pin/unpin molecule
gt mol step done <step>  # Complete step
```

## Best Practices

1. Use `--continue` for momentum
2. Check `bd mol current` before resuming
3. Squash completed molecules for audit trail
4. Burn routine wisps
