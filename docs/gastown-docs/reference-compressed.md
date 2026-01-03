# Gas Town Reference

## Directory Structure

```
~/gt/                       Town root
├── .beads/                 Town-level beads (hq-*)
├── mayor/town.json
└── <rig>/                  Project container (NOT git clone)
    ├── config.json         Rig identity
    ├── .repo.git/          Bare repo (shared by worktrees)
    ├── mayor/rig/          Mayor clone (canonical beads)
    ├── refinery/rig/       Worktree on main
    ├── witness/            No clone (monitors only)
    ├── crew/<name>/        Human workspaces
    └── polecats/<name>/    Worker worktrees
```

## Beads Routing

```bash
bd show gp-xyz    # Routes to greenplace rig
bd show hq-abc    # Routes to town-level
# Debug: BD_DEBUG_ROUTING=1 bd show <id>
```

## Configuration

**Rig** (`config.json`): `{name, git_url, beads: {prefix}}`
**Settings** (`settings/config.json`): theme, max_workers, merge_queue
**Runtime** (`.runtime/`): PIDs, ephemeral (gitignored)

## Formula Format

```toml
formula = "name"
type = "workflow"
[[steps]]
id = "step-id"
title = "{{var}}"
needs = ["dependency"]
```

## Molecule Lifecycle

```
Formula → bd cook → Proto → bd mol pour → Mol → bd squash → Digest
                         └→ bd mol wisp → Wisp → bd burn → (gone)
```

## Key Commands

**Town:** `gt install`, `gt doctor [--fix]`
**Rig:** `gt rig add/list/remove`
**Convoy:** `gt convoy list/status/create` (primary dashboard)
**Work:** `gt sling <bead> <rig>` (auto-creates convoy)
**Mail:** `gt mail inbox/read/send`
**Escalation:** `gt escalate [-s CRITICAL|HIGH|MEDIUM] "msg"`
**Sessions:** `gt handoff`, `gt peek`, `gt nudge`, `gt seance`

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `BD_ACTOR` | Agent identity |
| `BEADS_NO_DAEMON` | Required for worktree polecats |
| `GT_ROLE` | Agent role (mayor, polecat, etc.) |
| `GT_RIG` | Rig name |

## Patrol Agents

| Agent | Molecule | Job |
|-------|----------|-----|
| Deacon | mol-deacon-patrol | Lifecycle, health |
| Witness | mol-witness-patrol | Monitor polecats |
| Refinery | mol-refinery-patrol | Merge queue |

## Architecture

- Bare repo pattern: `.repo.git/` shared by worktrees
- Beads as control plane: molecule steps ARE beads issues
- Nondeterministic idempotence: any worker can continue any molecule
- Convoy tracking: persistent. Swarm: ephemeral workers on convoy issues
