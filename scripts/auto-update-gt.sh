#!/bin/bash
# Auto-update Gas Town (GT) - runs weekly on Sunday at midnight
# Syncs fork with upstream (steveyegge/gastown), rebuilds binary
#
# Components:
#   Timer:   /etc/systemd/system/gt-update.timer
#   Service: /etc/systemd/system/gt-update.service
#   Script:  This file

set -euo pipefail

LOG_FILE="$HOME/.gastown/logs/auto-update.log"
GT_DIR="/home/dev/Dropbox/Projects/sai-gas"
DOC_FILE="$HOME/Dropbox/Projects/sai/.claude-shared/docs/architecture/automation.md"
UPSTREAM_REPO="steveyegge/gastown"
MAX_RETRIES=3
RETRY_DELAY=5

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Retry function with exponential backoff
retry_cmd() {
    local cmd="$1"
    local description="$2"
    local attempt=1
    local delay=$RETRY_DELAY

    while [ $attempt -le $MAX_RETRIES ]; do
        log "  Attempt $attempt/$MAX_RETRIES: $description"
        if output=$(eval "$cmd" 2>&1); then
            echo "$output"
            return 0
        fi
        log "  Failed (attempt $attempt): ${output:-no output}"
        if [ $attempt -lt $MAX_RETRIES ]; then
            log "  Retrying in ${delay}s..."
            sleep $delay
            delay=$((delay * 2))
        fi
        attempt=$((attempt + 1))
    done
    return 1
}

log "=== Starting Gas Town (GT) auto-update check ==="

# 1. Get current version
cd "$GT_DIR"
# gt version outputs: "gt version 0.2.1 (dev: main@f301c085bdee)"
CURRENT=$("$GT_DIR/gt" version 2>/dev/null | grep -oP 'version \K[0-9.]+' | head -1 || echo "0.0.0")
log "Current version: v$CURRENT"

# 2. Fetch upstream
log "Fetching upstream..."
if ! git remote | grep -q upstream; then
    log "Adding upstream remote..."
    git remote add upstream "https://github.com/$UPSTREAM_REPO.git"
fi

retry_cmd "git fetch upstream" "Fetch upstream" || {
    log "ERROR: Failed to fetch upstream after $MAX_RETRIES attempts"
    exit 1
}

# 3. Get latest tag from upstream
LATEST=$(git describe --tags --abbrev=0 upstream/main 2>/dev/null || echo "")
if [ -z "$LATEST" ]; then
    # No tags, use commit hash comparison
    LOCAL_COMMIT=$(git rev-parse HEAD)
    UPSTREAM_COMMIT=$(git rev-parse upstream/main)

    if [ "$LOCAL_COMMIT" = "$UPSTREAM_COMMIT" ]; then
        log "Already up to date with upstream (commit: ${LOCAL_COMMIT:0:7})"
        exit 0
    fi

    log "New commits available from upstream"
    LATEST="upstream/main"
else
    log "Latest upstream tag: $LATEST"
fi

# 4. Compare versions (strip 'v' prefix from tag for comparison)
LATEST_NUM=$(echo "$LATEST" | sed 's/^v//')
if [ "$CURRENT" = "$LATEST_NUM" ]; then
    log "Already up to date (v$CURRENT). No action needed."
    exit 0
fi

log "Update available: v$CURRENT -> $LATEST"

# 5. Check for local changes
if ! git diff --quiet || ! git diff --cached --quiet; then
    log "WARNING: Local changes detected. Stashing..."
    git stash push -m "auto-update-stash-$(date +%Y%m%d%H%M%S)"
fi

# 6. Backup current binary
cp "$GT_DIR/gt" "/tmp/gt.backup.$(date +%Y%m%d%H%M%S)" 2>/dev/null || true

# 7. Sync with upstream (rebase our changes on top)
log "Rebasing on upstream/main..."
if ! git rebase upstream/main; then
    log "ERROR: Rebase failed - conflicts detected"
    log "  Run manually: cd $GT_DIR && git rebase --abort && git status"
    git rebase --abort
    exit 1
fi

# 8. Build new binary
log "Building new binary..."
if ! go mod tidy; then
    log "ERROR: go mod tidy failed"
    exit 1
fi

if ! go build -o gt ./cmd/gt; then
    log "ERROR: Build failed"
    log "  Restoring backup binary..."
    cp "/tmp/gt.backup."* "$GT_DIR/gt" 2>/dev/null || true
    exit 1
fi

# 9. Verify new binary
NEW_VERSION=$("$GT_DIR/gt" version 2>/dev/null | grep -oP 'version \K[0-9.]+' | head -1 || echo "unknown")
log "New binary version: v$NEW_VERSION"

if ! "$GT_DIR/gt" status &>/dev/null; then
    log "WARNING: gt status check returned non-zero (may be expected)"
fi

# 10. Push to origin if we have changes
COMMITS_AHEAD=$(git rev-list --count origin/main..HEAD 2>/dev/null || echo "0")
if [ "$COMMITS_AHEAD" -gt "0" ]; then
    log "Pushing $COMMITS_AHEAD commits to origin..."
    retry_cmd "git push origin main" "Push to origin" || {
        log "WARNING: Failed to push to origin (may need manual push)"
    }
fi

# 11. Update documentation
if [ -f "$DOC_FILE" ]; then
    log "Checking documentation..."
    TODAY=$(date '+%Y-%m-%d')

    # Update the automation doc if needed
    if grep -q "Gas Town" "$DOC_FILE"; then
        log "Documentation already includes GT auto-update"
    else
        log "NOTE: Consider adding GT auto-update to automation.md"
    fi
fi

# 12. Cleanup old backups (keep last 3)
ls -t /tmp/gt.backup.* 2>/dev/null | tail -n +4 | xargs rm -f 2>/dev/null || true

log "=== Update complete: v$CURRENT -> v$NEW_VERSION ==="
