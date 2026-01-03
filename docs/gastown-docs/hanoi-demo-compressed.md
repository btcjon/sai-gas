# Towers of Hanoi Demo

Durability proof: Gas Town executes arbitrarily long sequential workflows with crash recovery.

## What It Proves

- Large molecule creation (1000+ issues)
- Sequential execution with chained dependencies
- Crash recovery (resume after restart)
- Nondeterministic idempotence

## Scale

| Disks | Moves | Formula | Runtime |
|-------|-------|---------|---------|
| 7 | 127 | ~19 KB | ~14 sec |
| 10 | 1,023 | ~149 KB | ~2 min |
| 15 | 32,767 | ~4.7 MB | ~1 hour |

## Running

```bash
# Create wisp
WISP=$(bd mol wisp towers-of-hanoi-10 --json | jq -r '.new_epic_id')

# Get child IDs
bd list --parent=$WISP --limit=2000 --json | jq -r '.[].id' > /tmp/ids.txt

# Execute (serial close)
while read id; do bd --no-daemon close "$id" >/dev/null 2>&1; done < /tmp/ids.txt

# Cleanup
bd mol burn $WISP --force
```

## Why Wisps

- No git pollution (don't sync to JSONL)
- Auto-cleanup (burn without tombstones)
- Speed (no export overhead)
- `bd ready` excludes wisps by design

## Key Notes

- Dependencies via `needs` create proper `blocks` relations
- Close speed: ~109ms each, ~9/second serial
- Parent-child is hierarchy only, doesn't block execution

## Session Cycling

```bash
# Session 1
WISP=$(bd mol wisp towers-of-hanoi-10 --json | jq -r '.new_epic_id')
gt handoff -s "Hanoi demo" -m "Wisp: $WISP, progress: 400/1025"

# Session 2
bd list --parent=$WISP --status=open --limit=2000 --json | jq -r '.[].id' > /tmp/ids.txt
# Continue closing...
```

The molecule IS the state. No session memory needed.
