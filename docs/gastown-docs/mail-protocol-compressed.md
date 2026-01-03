# Mail Protocol

Inter-agent communication via `type=message` beads routed by `gt mail`.

## Message Types

| Type | Route | Purpose |
|------|-------|---------|
| `POLECAT_DONE` | Polecat → Witness | Signal completion, trigger cleanup |
| `MERGE_READY` | Witness → Refinery | Branch ready for merge queue |
| `MERGED` | Refinery → Witness | Merge success, safe to nuke polecat |
| `MERGE_FAILED` | Refinery → Witness | Non-conflict failure (tests/build) |
| `REWORK_REQUEST` | Refinery → Witness | Conflicts, need rebase |
| `WITNESS_PING` | Witness → Deacon | Second-order monitoring |
| `HELP` | Any → Mayor | Request intervention |
| `HANDOFF` | Agent → self | Session continuity |

## Body Formats

**POLECAT_DONE:**
```
Exit: MERGED|ESCALATED|DEFERRED
Issue: <issue-id>
MR: <mr-id>
Branch: <branch>
```

**MERGE_READY/MERGED:**
```
Branch: <branch>
Issue: <issue-id>
Polecat: <polecat-name>
```

**HANDOFF:**
```
attached_molecule: <mol-id>
## Context
<notes>
## Next
<what to do>
```

## Flows

**Completion:** Polecat → POLECAT_DONE → Witness (verify) → MERGE_READY → Refinery (merge) → MERGED → Witness (nuke)

**Failure:** Refinery → MERGE_FAILED → Witness → Polecat (rework)

**Conflict:** Refinery → REWORK_REQUEST → Witness → Polecat (rebase) → POLECAT_DONE → retry

## Addresses

```
greenplace/witness           # Rig role
greenplace/polecats/nux      # Specific polecat
mayor/                       # Town-level
```

## Commands

```bash
gt mail send <addr> -s "Subject" -m "Body"
gt mail inbox
gt mail read <id>
gt mail ack <id>
```

## Conventions

- Subject: `TYPE_NAME <context>` (uppercase prefix)
- Body: Key-value pairs, blank line, then freeform markdown
