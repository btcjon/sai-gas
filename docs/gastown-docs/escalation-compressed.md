# Escalation Protocol

Agents escalate when automated resolution fails. Routes through tiers: Worker → Deacon → Mayor → Overseer.

## Severity Levels

| Level | Priority | Description |
|-------|----------|-------------|
| CRITICAL | P0 | System-threatening (data corruption, security breach) |
| HIGH | P1 | Important blocker (merge conflict, critical bug) |
| MEDIUM | P2 | Standard (design decision, unclear requirements) |

## Categories

| Category | Route | Use Case |
|----------|-------|----------|
| `decision` | Deacon → Mayor | Multiple valid paths |
| `help` | Deacon → Mayor | Need guidance |
| `blocked` | Mayor | Unresolvable dependency |
| `failed` | Deacon | Unexpected error |
| `emergency` | Overseer (direct) | Security/data integrity |
| `gate_timeout` | Deacon | Gate didn't resolve |
| `lifecycle` | Witness | Worker stuck |

## Commands

```bash
# Basic
gt escalate "Database migration failed"
gt escalate -s CRITICAL "Data corruption detected"

# Category-based
gt escalate --type decision "Which auth approach?"
gt escalate --type emergency "Security vulnerability"

# Explicit routing
gt escalate --to mayor "Cross-rig coordination"
gt escalate --forward --to mayor "Deacon couldn't resolve"

# Structured decision
gt escalate --type decision \
  --question "Which auth approach?" \
  --options "JWT,Session cookies,OAuth2" \
  --context "Admin panel needs login" \
  --issue bd-xyz
```

## Tiered Flow

```
Worker → gt escalate
    ↓
Deacon ─► Can resolve? → Update issue, re-sling
    │
    └► Cannot? → gt escalate --forward --to mayor
                     ↓
                Mayor → Same pattern → Overseer
```

## Decision Pattern

`--type decision` updates issue with structured format:

```markdown
## Decision Needed
**Question:** Which auth approach?
| Option | Description |
| A | JWT tokens |
| B | Session cookies |
**To resolve:** Comment with choice, reassign to worker
```

## Integration

- **Gate timeouts**: Witness escalates when timer gates expire
- **Stuck polecats**: Witness escalates lifecycle issues
- **Merge failures**: Refinery escalates failed merges

## Polecat Exit Pattern

```bash
bd update $ISSUE --notes "## Decision Needed..."
gt escalate --type decision --issue $ISSUE "Needs decision"
gt done --exit ESCALATED
```

## When to Escalate

**Do:** System errors, security issues, unresolvable conflicts, ambiguous requirements, design decisions, stuck loops, gate timeouts

**Don't:** Normal workflow, recoverable errors, answerable from context

## Viewing

```bash
bd list --status=open --tag=escalation
bd close <id> --reason "Resolved by fixing X"
```
