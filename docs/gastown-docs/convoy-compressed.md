# Convoys

Persistent tracking units for batched work across rigs.

## Commands

```bash
gt convoy create "Feature X" gt-abc gt-def --notify overseer  # Create
gt convoy status hq-cv-abc                                     # Check progress
gt convoy list                                                 # Dashboard (active)
gt convoy list --all                                           # Include landed
```

## Concept

```
            🚚 Convoy (hq-cv-abc)
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
   [gt-xyz]    [gt-def]    [bd-abc]
       │            │            │
       ▼            ▼            ▼
   [polecat]   [polecat]   [polecat]
              "the swarm"
```

| Concept | Persistent | Description |
|---------|------------|-------------|
| **Convoy** | Yes (hq-cv-*) | Tracking unit you create/monitor |
| **Swarm** | No | Ephemeral workers on convoy issues |

## Lifecycle

```
OPEN ──(all issues close)──► CLOSED
  ↑                              │
  └──(add issues: auto-reopens)──┘
```

## Usage

**Create:**
```bash
gt convoy create "Deploy v2.0" gt-abc bd-xyz --notify gastown/joe
gt convoy create "Fix auth" gt-auth-fix  # Single issue OK
```

**Add issues** (direct bd command):
```bash
bd dep add hq-cv-abc gt-new-issue --type=tracks
```

**Status output:**
```
🚚 hq-cv-abc: Deploy v2.0
  Status:    ●
  Progress:  2/4 completed
  ✓ gt-xyz: Update API endpoint
  ✓ bd-abc: Fix validation
  ○ bd-ghi: Update docs
```

## Auto-Convoy

`gt sling bd-xyz beads/amber` auto-creates convoy "Work: bd-xyz" for dashboard visibility.

## Cross-Rig Tracking

Convoys (hq-cv-*) track issues from any rig. The `tracks` relation is non-blocking and additive.

```bash
gt convoy create "Full-stack" gt-frontend-abc gt-backend-def bd-docs-xyz
```

## Notifications

```bash
gt convoy create "X" gt-abc --notify gastown/joe --notify mayor/
```

Landing notification includes: issue list, duration, completion status.

## Convoy vs Rig Status

| `gt convoy status` | Cross-rig batch progress |
| `gt rig status` | Single rig worker activity |
