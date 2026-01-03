# Federation Architecture

Multi-workspace coordination for Gas Town and Beads.

## Why Federation

Real projects span multiple repos: microservices, platform teams, contractors, acquisitions. Federation provides unified tracking while preserving team autonomy.

## Entity Model

```
Level 1: Entity    - Person or organization
Level 2: Chain     - Workspace/town per entity
Level 3: Work Unit - Issues, tasks, molecules
```

## URI Scheme

```
hop://entity/chain/rig/issue-id           # Full HOP reference
hop://steve@example.com/main-town/gp/gp-xyz

beads://platform/org/repo/issue-id        # Cross-repo (same platform)
beads://github/acme/backend/ac-123

gp-xyz                                     # Local (prefix routes)
greenplace/gp-xyz                          # Different rig, same chain
```

## Relationship Types

**Employment:**
```json
{"type": "employment", "entity": "alice@example.com", "organization": "acme.com"}
```

**Cross-Reference:**
```json
{"references": [{"type": "depends_on", "target": "hop://other/chain/rig/id"}]}
```

**Delegation:**
```json
{"type": "delegation", "parent": "hop://acme/proj-123", "child": "hop://alice/gp-xyz"}
```

## Agent Provenance

```bash
# Git commits
GIT_AUTHOR_NAME="greenplace/crew/joe"
GIT_AUTHOR_EMAIL="steve@example.com"  # Workspace owner

# Beads operations
BD_ACTOR="greenplace/crew/joe"
```

## Discovery

```bash
# Workspace metadata: ~/gt/.town.json
gt remote add acme hop://acme.com/engineering
bd show hop://acme.com/eng/ac-123    # Fetch remote issue
bd list --org=acme.com               # All org work
bd audit --actor=greenplace/crew/joe # Agent history
```

## Implementation Status

- [x] Agent identity in git commits
- [x] BD_ACTOR default in beads create
- [x] Workspace metadata (.town.json)
- [x] Cross-workspace URI scheme
- [ ] Remote registration
- [ ] Cross-workspace queries
- [ ] Delegation primitives

## Design Principles

1. Flat namespace - entities not nested, relationships connect them
2. Relationships over hierarchy - graph structure, not tree
3. Git-native - uses git mechanics (remotes, refs)
4. Incremental - works standalone, gains power with federation
5. Privacy-preserving - each entity controls visibility
