#!/usr/bin/env bash
# set-up-named-tunnel.sh — one-shot script that runs AFTER `cloudflared tunnel login`.
#
# Creates a named tunnel, patches the config file with the UUID, routes
# DNS for live.interdependent.tv to this tunnel, stops the quick tunnel,
# and starts the named tunnel in the background.
#
# Idempotent: re-running skips steps that are already done.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Defaults target interdependent.dev (the dummy/test domain).
# Override via env vars when we repeat this for interdependent.tv:
#   TUNNEL_NAME=interdependent-tv-live HOSTNAME=live.interdependent.tv ./set-up-named-tunnel.sh
TUNNEL_NAME="${TUNNEL_NAME:-interdependent-dev-live}"
# NOTE: cannot use $HOSTNAME — that's a shell-reserved variable on macOS/bash
# that holds the machine's hostname (e.g. Christophers-Mac-mini.local). Using
# it here silently overrides our default and creates the wrong DNS record.
TUNNEL_HOSTNAME="${TUNNEL_HOSTNAME:-live.interdependent.dev}"
CFG="${HERE}/cloudflared-config.yml"
CRED_DIR="${HOME}/.cloudflared"
CERT="${CRED_DIR}/cert.pem"

# ── 0. Prerequisite: the origin cert from `cloudflared tunnel login` ────────
if [[ ! -f "${CERT}" ]]; then
  cat >&2 <<MSG
Missing ${CERT}. Run this first (one-time, opens a browser):

    cloudflared tunnel login

Then re-run this script.
MSG
  exit 2
fi

# ── 1. Create the tunnel (idempotent — reuse if it exists) ──────────────────
# `cloudflared tunnel list` prints one line per tunnel in the format
# "<uuid>  <name>  <created>  <connections>". We grep by exact name.
UUID="$(cloudflared tunnel list -o json 2>/dev/null \
  | python3 -c "import sys,json; d=json.load(sys.stdin) or []; [print(t['id']) for t in d if t['name']=='${TUNNEL_NAME}']")"

if [[ -z "${UUID}" ]]; then
  echo "Creating tunnel '${TUNNEL_NAME}'…"
  cloudflared tunnel create "${TUNNEL_NAME}"
  UUID="$(cloudflared tunnel list -o json \
    | python3 -c "import sys,json; d=json.load(sys.stdin) or []; [print(t['id']) for t in d if t['name']=='${TUNNEL_NAME}']")"
fi
echo "tunnel UUID: ${UUID}"

# ── 2. Patch the template config with the UUID ──────────────────────────────
# sed -i '' is the macOS form.
sed -i '' "s|<TUNNEL-UUID>|${UUID}|g" "${CFG}" || true

# ── 3. Route DNS — adds a CNAME record in Cloudflare pointing
#      live.interdependent.tv → <uuid>.cfargotunnel.com ─────────────────────
# This works because the domain is on Cloudflare DNS (moved from Route 53).
# If the CNAME already exists, cloudflared prints a harmless warning.
echo "Routing DNS ${TUNNEL_HOSTNAME} → tunnel…"
cloudflared tunnel route dns "${TUNNEL_NAME}" "${TUNNEL_HOSTNAME}" || true

# ── 4. Stop the quick tunnel (if running) ───────────────────────────────────
if pgrep -f 'cloudflared tunnel --url http://localhost:8888' >/dev/null; then
  echo "Stopping quick tunnel…"
  pkill -f 'cloudflared tunnel --url http://localhost:8888' || true
  sleep 1
fi

# ── 5. Start the named tunnel in the background ─────────────────────────────
# For production, use the launchd plist instead. This is for a quick test.
echo "Starting named tunnel in background…"
nohup cloudflared tunnel --config "${CFG}" run "${TUNNEL_NAME}" \
  > "${HERE}/cloudflared.log" 2>&1 &
NEW_PID=$!
echo "named tunnel pid ${NEW_PID}"

# ── 6. Wait for the tunnel to register, then prove it routes ───────────────
echo "Waiting for tunnel to register with Cloudflare edge…"
for i in {1..40}; do
  if grep -q "Registered tunnel connection" "${HERE}/cloudflared.log" 2>/dev/null; then
    echo "  ✓ registered after ${i}×0.5s"
    break
  fi
  sleep 0.5
done

# ── 7. Verify the hostname resolves + the manifest serves ──────────────────
echo
echo "── smoke test ──"
if ! dig +short "${TUNNEL_HOSTNAME}" | grep -q .; then
  echo "  ⚠ DNS for ${TUNNEL_HOSTNAME} not yet visible to this resolver."
  echo "    Cloudflare propagation is usually <30s. Try:"
  echo "      dig @1.1.1.1 ${TUNNEL_HOSTNAME}"
  echo "    If still empty after a minute, check the DNS → Records tab in the Cloudflare dashboard."
else
  curl -fsS "https://${TUNNEL_HOSTNAME}/program/index.m3u8" | head -3 \
    && echo "  ✓ manifest served via named tunnel"
fi

echo
echo "── next step ──"
echo "  For permanent autostart, install the launchd agent:"
echo "    cp ${HERE}/tv.interdependent.cloudflared.plist ~/Library/LaunchAgents/"
echo "    launchctl load  ~/Library/LaunchAgents/tv.interdependent.cloudflared.plist"
