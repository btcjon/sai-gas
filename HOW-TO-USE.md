# How Gas Town Works

> Last updated: 2026-01-02 | Status: active

Gas Town is a system for managing AI agents that do coding work. Think of it like a factory where workers (called "polecats") each get their own workspace and tasks.

---

## The Basic Idea

1. **You** are the Mayor - you assign work
2. **Polecats** are workers - they do the work in isolated workspaces
3. **Beads** are tasks - like tickets or issues
4. **Hooks** are assignments - when a worker has something on their hook, they work on it

---

## Quick Start (5 Minutes)

### 1. Check Everything is Running

```bash
cd /home/dev
gt status
```

You should see your "town" with rigs listed.

### 2. See Available Tasks

```bash
cd /home/dev/sai/mayor/rig
bd list --status=open
```

### 3. Create a Task

```bash
bd create -t "Fix the login bug" -b "Users can't log in on mobile"
```

Write down the ID it gives you (like `sai-abc123`).

### 4. Create a Worker

```bash
gt polecat add sai/LoginFixer
```

### 5. Give the Worker the Task

```bash
gt sling sai-abc123 sai/LoginFixer --naked
```

### 6. Check Progress

```bash
gt polecat status sai/LoginFixer
```

### 7. When Done, Clean Up

```bash
bd close sai-abc123 -m "Fixed the bug"
gt polecat rm sai/LoginFixer
```

---

## Key Commands Cheat Sheet

| What You Want | Command |
|---------------|---------|
| See town status | `gt status` |
| List all tasks | `bd list` |
| Create a task | `bd create -t "Title" -b "Description"` |
| Create a worker | `gt polecat add <rig>/<name>` |
| Assign task to worker | `gt sling <task-id> <rig>/<worker>` |
| Check worker status | `gt polecat status <rig>/<worker>` |
| Close a task | `bd close <task-id> -m "Done message"` |
| Remove a worker | `gt polecat rm <rig>/<worker>` |
| Check your mail | `gt mail inbox` |
| Send mail | `gt mail send <address> -s "Subject" -m "Message"` |

---

## Understanding the Folders

```
/home/dev/                    # "Town" - everything lives here
├── sai/                      # A "rig" (project container)
│   ├── mayor/rig/            # Where you manage tasks
│   ├── polecats/             # Workers live here
│   │   └── LoginFixer/       # Each worker gets own folder
│   └── crew/                 # Long-term workers (optional)
```

---

## The Propulsion Principle

The core rule of Gas Town:

> **If a worker finds something on their hook, they run it immediately.**

No asking permission. No waiting. When work is assigned, work begins.

This is why Gas Town is fast - agents don't wait around.

---

## Common Workflows

### Fix a Bug

```bash
# 1. Create the task
bd create -t "Fix bug X" -b "Details here"

# 2. Create worker and assign
gt polecat add sai/BugFixer
gt sling sai-XXXXX sai/BugFixer --naked

# 3. Worker does the work...

# 4. Close when done
bd close sai-XXXXX -m "Fixed"
gt polecat rm sai/BugFixer
```

### Work on Multiple Things at Once

```bash
# Create multiple workers
gt polecat add sai/Worker1
gt polecat add sai/Worker2

# Give each a task
gt sling task-1 sai/Worker1 --naked
gt sling task-2 sai/Worker2 --naked

# Track with convoy
gt convoy create "Sprint work" task-1 task-2
gt convoy list
```

### Check What's Happening

```bash
gt status              # Overall view
gt convoy list         # All active work batches
gt polecat list sai    # Workers in SAI rig
gt mail inbox          # Messages
```

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "command not found" | Make sure gt and bd are in PATH |
| "prefix mismatch" | Run commands from `/home/dev/sai/mayor/rig` |
| Worker stuck | Check `gt polecat status <worker>` |
| Can't create convoy | Town beads need `hq` prefix |

---

## Learn More

- Full documentation: `docs/` folder in this repo
- Understanding roles: `docs/understanding-gas-town.md`
- The propulsion philosophy: `docs/propulsion-principle.md`
