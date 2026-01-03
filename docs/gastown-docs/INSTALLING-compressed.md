# Installing Gas Town

## Prerequisites

| Required | Version | Check |
|----------|---------|-------|
| Go | 1.24+ | `go version` |
| Git | 2.20+ | `git --version` |
| Beads | latest | `bd version` |

**Optional (Full Stack):** tmux 3.0+, Claude Code

## Install

```bash
# macOS
brew install go git tmux

# Linux (Debian/Ubuntu)
sudo apt install -y git
# Go: use go.dev installer (apt may be outdated)
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
```

```bash
# Install binaries
go install github.com/steveyegge/gastown/cmd/gt@latest
go install github.com/steveyegge/beads/cmd/bd@latest

# Create workspace
gt install ~/gt

# Add a project
gt rig add myproject https://github.com/you/repo.git

# Verify
gt doctor
gt status
```

## Modes

**Minimal (No Daemon):** Manual Claude instances, Gas Town tracks state only
```bash
gt convoy create "Fix bugs" issue-123
cd ~/gt/myproject/polecats/<worker> && claude --resume
```

**Full Stack (Daemon):** Agents run in tmux, auto-managed
```bash
gt daemon start
gt convoy create "Feature X" issue-123
gt sling issue-123 myproject
gt mayor attach  # monitor
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `gt: command not found` | Add `$HOME/go/bin` to PATH |
| `bd: command not found` | `go install .../beads/cmd/bd@latest` |
| Doctor errors | `gt doctor --fix` |
| Daemon not starting | Check tmux: `tmux -V` |
| Git auth issues | Test: `ssh -T git@github.com` |
| Beads sync issues | `bd sync --status`, `bd doctor` |

## Update

```bash
go install github.com/steveyegge/gastown/cmd/gt@latest
go install github.com/steveyegge/beads/cmd/bd@latest
gt doctor --fix
```
