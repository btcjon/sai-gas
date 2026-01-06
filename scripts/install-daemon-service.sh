#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_FILE="$SCRIPT_DIR/../systemd/gt-daemon.service"
DEST="/etc/systemd/system/gt-daemon.service"

echo "Installing GT daemon service..."

if [[ ! -f "$SERVICE_FILE" ]]; then
    echo "Error: Service file not found at $SERVICE_FILE"
    exit 1
fi

# Stop existing service if running
if systemctl is-active --quiet gt-daemon 2>/dev/null; then
    echo "Stopping existing gt-daemon service..."
    sudo systemctl stop gt-daemon
fi

sudo cp "$SERVICE_FILE" "$DEST"
sudo systemctl daemon-reload
sudo systemctl enable gt-daemon
sudo systemctl start gt-daemon

echo ""
echo "GT daemon service installed and started!"
echo ""
systemctl status gt-daemon --no-pager
