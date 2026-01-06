# Self-Improving Skills System

A reflection-based approach to persistent session learning for Claude Code.

## The Problem

LLMs do not persist memory between sessions. Every correction made in one session is forgotten in the next. Users repeat themselves endlessly:

- "Use tabs, not spaces" (session 1)
- "Use tabs, not spaces" (session 2)
- "Use tabs, not spaces" (session 47)

Even with CLAUDE.md files and skills, manual updates require discipline and interruption.

## The Solution

A "reflect" skill that:

1. Analyzes the current session for corrections and patterns
2. Extracts learnings with confidence levels (high/medium/low)
3. Updates skill files with new rules and patterns
4. Optionally versions everything in Git

The key insight: **Corrections are signals.** When a user corrects the agent, that correction likely applies to future sessions too.

## How It Works

### Signal Detection

The reflect skill scans conversations for two signal types:

| Signal | Pattern | Example |
|--------|---------|---------|
| **Corrections** | User contradicts or redirects agent | "No, use yarn not npm" |
| **Approvals** | User confirms approach worked | "Perfect, that's exactly right" |

Corrections indicate what to change. Approvals confirm what to preserve.

### Confidence Classification

Each extracted learning gets a confidence level:

| Level | Criteria | Action |
|-------|----------|--------|
| **HIGH** | Explicit rule statement | Auto-update skill |
| **MEDIUM** | Pattern that worked well | Propose update |
| **LOW** | Observation worth noting | Log for review |

**High confidence triggers:**
- "Never do X"
- "Always use Y"
- "From now on, Z"
- Repeated corrections (same thing, multiple times)

**Medium confidence triggers:**
- Positive feedback on approach
- Successful pattern completion
- User acceptance without explicit praise

**Low confidence triggers:**
- Implicit preferences
- Context-specific decisions
- One-time instructions

### Manual Flow

1. User works with Claude, makes corrections during session
2. User calls `/reflect` command
3. Reflect skill scans conversation
4. Proposes changes with confidence levels
5. User reviews: accept, modify, or reject
6. If accepted: edits skill file, commits to Git

### Automatic Flow

1. Configure stop hook to trigger reflection
2. When session ends, hook analyzes conversation automatically
3. High-confidence learnings update skills automatically
4. Medium/low-confidence learnings queued for review

## Relation to Existing Systems

| System | Purpose | Trigger |
|--------|---------|---------|
| **reflect** | Real-time session learning | Manual command or session end |
| **compound-learnings** | Batch analysis of past handoffs | Periodic review |
| **agent-memory (mem0)** | Persistent fact storage | Automatic |

These systems complement each other:

- **reflect** catches corrections as they happen
- **compound-learnings** finds patterns across many sessions
- **agent-memory** stores facts and context

## Benefits

1. **"Correct once, never again"** - Fixes persist automatically
2. **No complex infrastructure** - Just markdown files and Git
3. **Human readable** - All learnings stored as plain text
4. **Auditable history** - Git shows exactly how the system learned
5. **Works with any skill** - Code review, API design, testing, docs

## Considerations

### Safety

The system should be conservative:
- High confidence for automatic updates only
- Manual review for medium/low confidence
- Easy rollback via Git history

### Toggle Control

Users need control over automatic reflection:
- `reflect on` - Enable automatic session-end reflection
- `reflect off` - Disable automatic reflection
- `reflect status` - Check current state

### Scope

Not every correction should become a permanent rule:
- Project-specific preferences vs universal patterns
- Temporary workarounds vs best practices
- User mood vs genuine preference

The reflect skill should distinguish between:
- "Use yarn in THIS project" (project-specific, goes to CLAUDE.md)
- "Use yarn for all projects" (universal, goes to user config)

## Source

Based on concepts from the Developers Digest video on self-improving agents.

## See Also

- `/docs/REFLECT-SKILL-IMPLEMENTATION.md` - Technical implementation details
- `compound-learnings` skill - Batch learning from handoffs
- `agent-memory` skill - mem0 integration
- `self-improvement.md` rule - Learning principles
