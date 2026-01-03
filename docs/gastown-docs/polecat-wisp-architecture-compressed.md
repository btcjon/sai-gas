# Polecat Wisp Architecture

Polecats execute work via hooked molecules, closing steps sequentially.

## Molecule Types

| Type | Storage | Use |
|------|---------|-----|
| Molecule | `.beads/` (synced) | Discrete deliverables, audit trail |
| Wisp | `.beads-wisp/` (ephemeral) | Patrol cycles, no accumulation |

Polecats use **molecules** (audit value). Patrol agents use **wisps**.

## Step Execution

```bash
# Propulsion approach (1 command)
bd mol current                    # Where am I?
# ... do the work ...
bd close gt-abc.3 --continue      # Close and auto-advance
```

The `--continue` flag: closes step → finds next ready → marks in_progress → outputs transition.

## Example Session

```
$ bd mol current
  ✓ gt-abc.1: Design
  → gt-abc.2: Implement [in_progress] <- YOU ARE HERE
  ○ gt-abc.3: Tests

$ bd close gt-abc.2 --continue
✓ Closed gt-abc.2: Implement
→ Marked gt-abc.3: Tests in_progress
```

## Molecule Completion

```bash
# After last step
bd mol squash gt-abc --summary "Implemented user auth"
# Or for routine: bd mol burn gt-abc
```

## Hook Management

```bash
gt hook                        # What's on my hook?
gt mail inbox                  # Check for assignments
gt mol attach-from-mail <id>   # Attach work from mail
gt done                        # Complete: sync, merge queue, notify Witness
```

## Workflow Summary

```
1. Spawn with work on hook
2. gt hook → bd mol current → Execute step
3. bd close <step> --continue
4. If more steps: GOTO 2
5. gt done
6. Wait for Witness cleanup
```

## Wisp vs Molecule

| Question | Molecule | Wisp |
|----------|----------|------|
| Audit trail needed? | Yes | No |
| Repeats continuously? | No | Yes |
| Discrete deliverable? | Yes | No |
