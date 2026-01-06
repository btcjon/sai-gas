#!/bin/bash
# Disable Full Ferrari mode - Return to manual GT operation
set -e

echo "=== Disabling Full Ferrari Mode ==="
echo ""

# 1. Stop systemd service if running
if systemctl is-active --quiet gt-daemon 2>/dev/null; then
    echo "[1/4] Stopping gt-daemon service..."
    sudo systemctl stop gt-daemon
    sudo systemctl disable gt-daemon
else
    echo "[1/4] gt-daemon service not running"
fi

# 2. Remove environment file
ENV_FILE="$HOME/.gt-ferrari.env"
if [[ -f "$ENV_FILE" ]]; then
    rm -f "$ENV_FILE"
    echo "[2/4] Removed $ENV_FILE"
else
    echo "[2/4] No environment file to remove"
fi

# 3. Unset current session variables
unset GT_AUTO_SLING
unset GT_AUTO_BEAD
unset GT_AUTO_CONVOY
unset GT_COMPUTE_ROUTING
unset GT_MIN_POLECAT_POOL
unset GT_MAX_POLECAT_POOL
unset GT_POLECAT_POOL_ENABLED
echo "[3/4] Unset environment variables"

# 4. Note about bashrc
echo "[4/4] Note: ~/.bashrc still has the source line (harmless if file doesn't exist)"

echo ""
echo "=== Full Ferrari Mode Disabled ==="
echo ""
echo "Manual operation restored."
echo "Start daemon manually:  gt daemon start"
echo "Re-enable with:         ./scripts/enable-full-ferrari.sh"
