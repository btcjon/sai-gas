# Agent Identity and Attribution

Every action, commit, and bead update is linked to a specific agent identity.

## BD_ACTOR Format

| Role | Format | Example |
|------|--------|---------|
| Mayor | `mayor` | `mayor` |
| Deacon | `deacon` | `deacon` |
| Witness | `{rig}/witness` | `gastown/witness` |
| Refinery | `{rig}/refinery` | `gastown/refinery` |
| Crew | `{rig}/crew/{name}` | `gastown/crew/joe` |
| Polecat | `{rig}/polecats/{name}` | `gastown/polecats/toast` |

## Attribution

**Git commits:**
```bash
GIT_AUTHOR_NAME="gastown/crew/joe"      # Agent (executor)
GIT_AUTHOR_EMAIL="steve@example.com"    # Owner (human)
# Result: abc123 Fix bug (gastown/crew/joe <steve@example.com>)
```

**Beads records:** `created_by`, `updated_by` from BD_ACTOR

**Events:** All events include `actor` field

## Environment (auto-set by daemon)

```bash
BD_ACTOR="gastown/polecats/toast"
GIT_AUTHOR_NAME="gastown/polecats/toast"
GT_ROLE="polecat"
GT_RIG="gastown"
GT_POLECAT="toast"
```

## Audit Queries

```bash
bd audit --actor=gastown/crew/joe    # All work by agent
bd audit --actor=gastown/*           # All work in rig
git log --author="gastown/crew/joe"  # Git history
```

## CV Model

```
steve@example.com          ← global identity (from git)
├── Town A                 ← workspace
│   └── gastown/crew/joe   ← agent executor
└── Town B
    └── acme/polecats/nux  ← agent executor
```

**Key points:**
- Global ID = email (already in git commits)
- Agents execute, humans own
- Polecats are ephemeral (like K8s pods)
- Skills derived from work evidence (files touched, tags, patterns)
- Multi-town aggregation via `bd cv email@example.com`

## Design Principles

1. Agents not anonymous - every action attributed
2. Work owned, not authored - agent creates, overseer owns
3. Attribution permanent - git preserves history
4. Format parseable - enables programmatic analysis
5. Consistent across systems - same format in git, beads, events
