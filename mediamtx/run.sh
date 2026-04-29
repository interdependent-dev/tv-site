#!/usr/bin/env bash
# run.sh — foreground launcher for local dev / manual broadcasts.
#
# For unattended / reboot-survivable operation, use the launchd plist instead:
#   cp tv.interdependent.mediamtx.plist ~/Library/LaunchAgents/
#   launchctl load  ~/Library/LaunchAgents/tv.interdependent.mediamtx.plist
#   launchctl start tv.interdependent.mediamtx
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${HERE}/mediamtx"
CFG="${HERE}/mediamtx.yml"

# Lazy install — makes `./run.sh` a one-command setup for a fresh checkout.
if [[ ! -x "${BIN}" ]]; then
  "${HERE}/install.sh"
fi

# exec, not plain invocation: signals (Ctrl-C, launchd SIGTERM) reach mediamtx
# directly, so it shuts down cleanly and releases its UDP sockets.
exec "${BIN}" "${CFG}"
