#!/usr/bin/env bash
# run.sh — build and run the mission control dashboard.
#
# Usage:
#   ./run.sh                 # foreground, ctrl-c to stop
#   CF_API_TOKEN=… ./run.sh  # enable Cloudflare Analytics panel
#
# Open: http://localhost:9998

set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${HERE}"

# Build a fresh binary every run — keeps things zero-friction.
go build -o ./dashboard-bin .

exec ./dashboard-bin
