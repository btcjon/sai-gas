# Reflect Skill Implementation

Technical details for implementing the self-improving skills system.

## Components

### 1. The Reflect Skill

Location: `.claude-shared/skills/reflect/SKILL.md`

The skill file contains instructions for:
- Scanning conversation for signals (corrections, approvals)
- Classifying confidence levels
- Proposing skill file updates
- Handling user approval flow

### 2. The /reflect Command

A slash command that invokes the skill:

```bash
# Reflect on current session
/reflect

# Target a specific skill
/reflect --skill code-review

# Show what would be learned without applying
/reflect --dry-run
```

### 3. Stop Hook Integration

For automatic reflection on session end:

```json
{
  "hooks": {
    "stop": [{
      "command": "~/.claude/scripts/reflect.sh",
      "timeout": 30000
    }]
  }
}
```

### 4. Git Integration

Version control for skill changes:

```bash
# After skill update
cd ~/.claude
git add skills/
git commit -m "reflect: learned [pattern description]"
git push
```

## Data Flow

```
Session Work
    |
    v
[/reflect command OR stop hook]
    |
    v
Conversation Analysis
    |
    v
Signal Extraction
    |
    +-> HIGH confidence -> Auto-update skill
    |
    +-> MEDIUM confidence -> Propose to user
    |
    +-> LOW confidence -> Log for later
```

## Conversation Analysis

### Pattern Detection

The reflect skill looks for linguistic patterns:

**Correction patterns:**
- "No, actually..."
- "That's not right..."
- "Use X instead of Y"
- "I said X, not Y"
- "Always do X" / "Never do Y"

**Approval patterns:**
- "Perfect"
- "That's exactly right"
- "Great job"
- "Keep doing it this way"

### Semantic Clustering

Group related corrections:

```
Correction 1: "Use tabs not spaces"
Correction 2: "Indent with tabs"
Correction 3: "Tab character for indentation"
    |
    v
Clustered Learning: "Use tabs for indentation"
```

## Skill File Update Format

When updating a skill file, append to a `## Learned Patterns` section:

```markdown
## Learned Patterns

### Pattern: Use tabs for indentation
- **Confidence:** HIGH
- **Source:** 3 corrections in session abc123
- **Learned:** 2024-01-15
- **Rule:** Use tab characters for all code indentation

### Pattern: Prefer yarn over npm
- **Confidence:** MEDIUM
- **Source:** User preference stated in session def456
- **Learned:** 2024-01-14
- **Rule:** Use yarn for package management unless project specifies otherwise
```

## Review Interface

For medium/low confidence learnings, present a review:

```
=== Proposed Learnings ===

[HIGH] Use tabs for indentation
  Source: "No, use tabs not spaces" (x3)
  Action: Will update code-style skill

[MEDIUM] Prefer functional components
  Source: "Let's use a functional component here"
  Action: Propose adding to react skill

[LOW] Consider error boundaries
  Source: Implicit in error handling discussion
  Action: Log for review

---
Accept all? [y/N/m for modify]
```

## Toggle Implementation

State file: `~/.claude/state/reflect-config.json`

```json
{
  "enabled": true,
  "autoApplyHighConfidence": true,
  "logLowConfidence": true,
  "targetDir": "~/.claude/skills"
}
```

Commands:
- `reflect on` - Sets `enabled: true`
- `reflect off` - Sets `enabled: false`
- `reflect status` - Shows current config

## Git Commit Message Format

```
reflect: [action] [pattern summary]

Confidence: [HIGH|MEDIUM|LOW]
Session: [session-id]
Skill: [skill-name]

[Optional: full pattern description]
```

Example:
```
reflect: learned indentation preference

Confidence: HIGH
Session: 2024-01-15-abc123
Skill: code-style

User corrected indentation style 3 times.
Now using tabs instead of spaces for all code.
```

## Rollback

If a learned pattern causes issues:

```bash
# View learning history
git log --oneline --grep="reflect:" ~/.claude/skills/

# Rollback specific learning
git revert <commit-hash>

# Or manually edit skill file to remove pattern
```

## Integration Points

| Component | Interacts With |
|-----------|----------------|
| reflect skill | Conversation history |
| /reflect command | Claude Code CLI |
| stop hook | Session lifecycle |
| Git | Skill file storage |
| compound-learnings | Can analyze reflect commits |

## Security Considerations

1. **Sanitize patterns** - Do not persist secrets or sensitive data
2. **Validate before write** - Check skill file syntax after modification
3. **Limit scope** - Only update skills user has permissions for
4. **Audit trail** - Git history provides accountability

## Example Implementation

### reflect.sh (stop hook)

```bash
#!/bin/bash

# Skip if disabled
CONFIG=~/.claude/state/reflect-config.json
if [ -f "$CONFIG" ]; then
  ENABLED=$(jq -r '.enabled' "$CONFIG")
  if [ "$ENABLED" != "true" ]; then
    exit 0
  fi
fi

# Invoke reflect analysis
claude --skill reflect --analyze-session

# High confidence updates happen automatically
# Medium/low are queued in ~/.claude/state/pending-learnings.json
```

### Skill file template

```markdown
---
name: reflect
description: Analyze session for learnings and update skills
---

# Reflect Skill

Analyzes the current session for corrections and patterns,
then proposes or applies skill file updates.

## Activation

Called via `/reflect` command or stop hook.

## Process

1. Scan conversation for correction/approval signals
2. Extract patterns with confidence classification
3. Cluster related patterns
4. For HIGH confidence: update skill files directly
5. For MEDIUM/LOW: propose changes for review

## Output

Returns structured learning report with:
- Patterns found
- Confidence levels
- Proposed changes
- Git commit (if changes applied)
```

## See Also

- `/docs/SELF-IMPROVING-SKILLS.md` - Concept overview
- `compound-learnings` skill - Batch learning approach
- `self-improvement.md` rule - Learning principles
