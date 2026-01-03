# The Propulsion Principle

> **If you find something on your hook, YOU RUN IT.**

Gas Town is a steam engine. Agents are pistons. When work is hooked, EXECUTE.

## Why

- No supervisor polling "did you start yet?"
- The hook IS your assignment
- Every delay stalls the engine
- Other agents may be blocked on YOUR output

## The Contract

1. You will find it on your hook
2. You will understand it (`bd show` / `gt hook`)
3. You will BEGIN IMMEDIATELY

## Propulsion Loop

```bash
bd close gt-abc.3 --continue   # One command: close + auto-advance
```

**Old way (3 commands, momentum lost):**
```bash
bd close gt-abc.3
bd ready --parent=gt-abc
bd update gt-abc.4 --status=in_progress
```

**New way (1 command, momentum preserved):**
```bash
bd close gt-abc.3 --continue
```

## Startup Behavior

1. `gt hook` - check hook
2. Work hooked → EXECUTE immediately
3. Hook empty → Check mail
4. Nothing → ERROR: escalate to Witness

## The Capability Ledger

Every completion is recorded. Your work is visible.
- Redemption is real (consistent work builds over time)
- Every completion = evidence autonomous execution works
- Your CV grows with every completion

Execute with care.
