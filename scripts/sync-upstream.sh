#!/bin/bash
# Sync fork with upstream while preserving local features
# Run from sai-gas directory

set -e

echo "=== SAI-GAS Upstream Sync ==="
echo "This will:"
echo "1. Save your features to a branch"
echo "2. Reset main to upstream"
echo "3. Cherry-pick your features back"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Commits to preserve (in dependency order)
KEEP_COMMITS=(
    # Compute subsystem
    "c645bd4"  # Phase 1 MVP
    "c2052d6"  # --compute flag
    "315eed5"  # Phase 3 witness monitoring
    "f753560"  # gt compute test
    "8ec8e69"  # Phase 2-3 integration

    # Convoy enhancements
    "5e4456d"  # auto-close 100%
    "8b7729f"  # --tree flag
    "75e9833"  # stranded detection
    "da01baa"  # merge command

    # Auto-leverage
    "cc481b1"  # phases 2-5
    "0f4ae26"  # auto-enable GT

    # Context/quota
    "cb8a513"  # usage-aware routing
    "a2ee3a1"  # gt context --usage
    "48fc04d"  # % left display

    # GT all + flight
    "0fcb2f8"  # gt all command
    "0e90ea3"  # preflight/postflight

    # Plugins
    "83cefa9"  # plugin commands
    "ca90533"  # merge-oracle
    "8608ec7"  # plan-oracle

    # Epic/templates
    "391b1f6"  # epic templates
    "79ace08"  # plan-to-epic

    # Misc essential
    "41a4175"  # GASTOWN_MODE env
    "c69b298"  # --json whoami
    "87a1e32"  # hooks check, batch sling
    "65adf97"  # agent monitoring
    "f193991"  # prevent double-nuke
)

DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_BRANCH="backup/pre-sync-$DATE"
FEATURES_BRANCH="sai-features"
BEADS_BACKUP="/tmp/beads-backup-$DATE"

echo ""
echo "Step 1: Creating backup branch..."
git branch "$BACKUP_BRANCH"
echo "  Created: $BACKUP_BRANCH"

echo ""
echo "Step 2: Backing up local files..."
cp -r .beads "$BEADS_BACKUP"
cp CLAUDE.md "$BEADS_BACKUP/CLAUDE.md.backup"
cp .gitignore "$BEADS_BACKUP/gitignore.backup"
cp AGENTS.md "$BEADS_BACKUP/AGENTS.md.backup" 2>/dev/null || true
echo "  Backed up to: $BEADS_BACKUP"
echo "  Issues: $(wc -l < .beads/issues.jsonl) lines"
echo "  MQ items: $(ls .beads/mq/ 2>/dev/null | wc -l)"
echo "  CLAUDE.md: $(wc -l < CLAUDE.md) lines"

echo ""
echo "Step 3: Fetching upstream..."
git fetch upstream

echo ""
echo "Step 4: Creating features branch from current main..."
git branch -D "$FEATURES_BRANCH" 2>/dev/null || true
git checkout -b "$FEATURES_BRANCH"

echo ""
echo "Step 5: Resetting main to upstream..."
git checkout main
git reset --hard upstream/main

echo ""
echo "Step 6: Restoring local files..."
# Restore beads data
cp "$BEADS_BACKUP/issues.jsonl" .beads/
cp "$BEADS_BACKUP/interactions.jsonl" .beads/ 2>/dev/null || true
cp "$BEADS_BACKUP/metadata.json" .beads/ 2>/dev/null || true
cp -r "$BEADS_BACKUP/mq" .beads/ 2>/dev/null || true
# Restore project files
cp "$BEADS_BACKUP/CLAUDE.md.backup" CLAUDE.md
cp "$BEADS_BACKUP/AGENTS.md.backup" AGENTS.md 2>/dev/null || true
# Append custom gitignore entries
echo "" >> .gitignore
echo "# SAI custom ignores" >> .gitignore
echo ".claude-shared" >> .gitignore
echo ".osgrep/" >> .gitignore
echo "  Restored: beads data, CLAUDE.md, AGENTS.md, .gitignore entries"

echo ""
echo "Step 7: Cherry-picking features..."
FAILED_COMMITS=()
for commit in "${KEEP_COMMITS[@]}"; do
    echo "  Cherry-picking $commit..."
    if ! git cherry-pick "$commit" --no-commit 2>/dev/null; then
        echo "    CONFLICT in $commit - skipping (will need manual review)"
        git cherry-pick --abort 2>/dev/null || true
        FAILED_COMMITS+=("$commit")
    else
        git commit -C "$commit" 2>/dev/null || true
    fi
done

echo ""
echo "Step 8: Rebuilding binary..."
go build -o gt ./cmd/gt

echo ""
echo "=== SYNC COMPLETE ==="
echo "Backup branch: $BACKUP_BRANCH"
echo "Features branch: $FEATURES_BRANCH (for reference)"
echo ""

if [ ${#FAILED_COMMITS[@]} -gt 0 ]; then
    echo "WARNING: These commits had conflicts and need manual review:"
    for commit in "${FAILED_COMMITS[@]}"; do
        echo "  $commit - $(git log --oneline -1 $commit 2>/dev/null || echo 'unknown')"
    done
    echo ""
    echo "To manually apply:"
    echo "  git cherry-pick <commit>"
    echo "  # resolve conflicts"
    echo "  git cherry-pick --continue"
fi

echo ""
echo "To push (FORCE REQUIRED):"
echo "  git push origin main --force"
