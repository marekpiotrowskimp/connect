#!/bin/bash
# Connect V3 start script for macOS
# Starts connectd service and launches the desktop UI.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$SCRIPT_DIR/bin"
CONNECTD="$BIN_DIR/connectd"
CONNECT_UI="$BIN_DIR/connect-ui"
LOG="/tmp/connectd.log"

# Prefer same-directory binaries; fall back to bin/
[ -x "./connectd" ] && CONNECTD="./connectd"
[ -x "./connect-ui" ] && CONNECT_UI="./connect-ui"

echo "Connect V3 — starting service..."

# Always restart connectd to ensure latest binary is used
pkill -x "connectd" 2>/dev/null || true
sleep 1

if [ ! -x "$CONNECTD" ]; then
    echo "connectd binary not found at $CONNECTD"
    echo "Run 'make build-connectd' first."
    exit 1
fi
nohup "$CONNECTD" > "$LOG" 2>&1 &
echo "connectd started (PID $!)"
sleep 1

if [ ! -x "$CONNECT_UI" ]; then
    echo "connect-ui binary not found at $CONNECT_UI"
    echo "Run 'make build-ui' first."
    exit 1
fi
echo "Launching connect-ui..."
open "$CONNECT_UI"
