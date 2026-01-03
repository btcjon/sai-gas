# Why These Features?

Enterprise AI challenges → Gas Town solutions.

## The Problem

AI agents write code, review PRs, fix bugs. Can't answer:
- Who did what? Which agent introduced this bug?
- Who's reliable? Which agents deliver quality?
- What's connected? Does this frontend depend on backend PR?

## The Solution: A Work Ledger

Work as structured data. Every action recorded. Every agent has a track record.

---

## Entity Tracking and Attribution

**Problem:** 50 agents, 10 projects, one bug. Which agent?
**Solution:** Distinct identity per agent.
```
Git: gastown/polecats/toast <owner@example.com>
Beads: created_by: gastown/crew/joe
```
**Why:** Debugging, compliance (SOX/GDPR), accountability.

---

## Work History (Agent CVs)

**Problem:** Complex Go refactor. 20 agents. Who's best?
**Solution:** Accumulated work history.
```bash
bd audit --actor=gastown/polecats/toast
bd stats --actor=gastown/polecats/toast --tag=go
```
**Why:** Performance management, capability matching, A/B testing models.

---

## Capability-Based Routing

**Problem:** Work in Go, Python, TypeScript. Manual assignment doesn't scale.
**Solution:** Skills from work history, automatic matching.
```bash
gt dispatch gt-xyz --prefer-skill=go
```

---

## Recursive Work Decomposition

**Problem:** Feature = 50 tasks across 8 repos. Flat lists don't capture structure.
**Solution:** Natural decomposition: Epic → Feature → Task. Roll-ups automatic.

---

## Cross-Project References

**Problem:** Frontend can't ship until backend lands. Different repos.
**Solution:** Explicit dependencies: `depends_on: beads://github/acme/backend/be-456`

---

## Federation

**Problem:** Multi-repo, multi-org visibility fragmented.
**Solution:** Federated workspaces.
```bash
gt remote add partner hop://partner.com/project
bd list --remote=partner
```

---

## Validation and Quality Gates

**Problem:** Agent says "done." Is it actually done?
**Solution:** Structured validation with attribution.
```json
{"validated_by": "gastown/refinery", "tests_passed": true}
```

---

## Real-Time Activity Feed

**Problem:** Multi-agent work is opaque.
**Solution:** `bd activity --follow` - work state as stream.

---

## Design Philosophy

1. Attribution not optional
2. Work is data (structured, queryable)
3. History matters (track records = trust)
4. Scale assumed (multi-repo, multi-agent, multi-org)
5. Verification over trust (quality gates are primitives)
