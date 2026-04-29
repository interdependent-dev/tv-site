#!/usr/bin/env bash
# smoke.sh — fast probe of the live INTERDEPENDENT TV stack.
#
# Validates that every running piece of the stack is reachable and
# returning sensible responses. Use after any deploy / launchd kick /
# config change to confirm the system is still healthy.
#
#   cd /Users/interdependent/tv && ./mediamtx/smoke.sh
#
# Exits 0 on full pass, 1 if any check fails. Each check is independent
# so a downstream failure doesn't mask upstream symptoms.
#
# Distinct from test.sh in this directory, which is an end-to-end test
# that brings up its own throwaway mediamtx instance + ffmpeg publisher.
# This script touches the *already running* services only — read-only.

set -u
PASS=0
FAIL=0
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

green()  { printf '\033[32m%s\033[0m' "$1"; }
red()    { printf '\033[31m%s\033[0m' "$1"; }
dim()    { printf '\033[90m%s\033[0m' "$1"; }

ok()   { echo "  $(green ✓) $1"; PASS=$((PASS+1)); }
fail() { echo "  $(red ✗) $1"; FAIL=$((FAIL+1)); }

section() { echo; echo "── $(dim "$1") ─────────────────────────────────"; }

# http_check LABEL URL [EXPECTED_CODE=200] [BODY_CONTAINS=]
http_check() {
  local label=$1 url=$2 expected_code=${3:-200} contains=${4:-}
  local code
  code=$(curl -s -o "$TMP" -w "%{http_code}" -m 6 "$url" 2>/dev/null || echo 000)
  if [[ "$code" != "$expected_code" ]]; then
    fail "$label  $(dim "[HTTP $code, want $expected_code]")"
    return
  fi
  if [[ -n "$contains" ]] && ! grep -q -- "$contains" "$TMP"; then
    fail "$label  $(dim "[body missing '$contains']")"
    return
  fi
  ok "$label"
}

tcp_listening() {
  local label=$1 port=$2
  if lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    ok "$label  $(dim "(TCP :$port)")"
  else
    fail "$label  $(dim "(TCP :$port — not listening)")"
  fi
}

udp_listening() {
  local label=$1 port=$2
  if lsof -nP -iUDP:"$port" >/dev/null 2>&1; then
    ok "$label  $(dim "(UDP :$port)")"
  else
    fail "$label  $(dim "(UDP :$port — not listening)")"
  fi
}

launchd_loaded() {
  local label=$1 agent=$2
  if launchctl list 2>/dev/null | grep -q "$agent"; then
    ok "$label  $(dim "($agent)")"
  else
    fail "$label  $(dim "($agent — not loaded)")"
  fi
}

# ─────────────────────────────────────────────────────────────────────
section "Listening ports"
tcp_listening "MediaMTX control API"  9997
tcp_listening "MediaMTX HLS"          8888
tcp_listening "MediaMTX WebRTC HTTP"  8889
tcp_listening "Mission Control"       9998
udp_listening "MediaMTX SRT ingest"   8890
udp_listening "MediaMTX WebRTC media" 8189

section "launchd agents"
launchd_loaded "mediamtx"     tv.interdependent.mediamtx
launchd_loaded "cloudflared"  tv.interdependent.cloudflared

section "MediaMTX control API"
http_check "GET /v3/paths/list"     "http://127.0.0.1:9997/v3/paths/list"     200 itemCount
http_check "GET /v3/srtconns/list"  "http://127.0.0.1:9997/v3/srtconns/list"  200 itemCount
http_check "GET /v3/hlsmuxers/list" "http://127.0.0.1:9997/v3/hlsmuxers/list" 200 itemCount

section "MediaMTX HLS egress"
# /program/index.m3u8 only exists when an OBS publisher is connected. The
# infrastructure passing without /program flowing is a normal "stack idle"
# state — only fail those checks when we know a publisher is connected.
PROGRAM_READY=$(curl -s -m 3 http://127.0.0.1:9997/v3/paths/list 2>/dev/null \
  | python3 -c "import json,sys; d=json.load(sys.stdin); print('1' if any(p['name']=='program' and p.get('ready') for p in d.get('items',[])) else '0')" 2>/dev/null \
  || echo 0)
if [[ "$PROGRAM_READY" == "1" ]]; then
  http_check "GET /program/index.m3u8" "http://127.0.0.1:8888/program/index.m3u8" 200 EXTM3U
else
  echo "  $(dim "(no /program publisher — skipping HLS egress + tunnel + frame checks)")"
fi

section "Mission Control dashboard"
http_check "GET /"                       "http://127.0.0.1:9998/"                       200 "MISSION CONTROL"
http_check "GET /app.css"                "http://127.0.0.1:9998/app.css"                200
http_check "GET /app.js"                 "http://127.0.0.1:9998/app.js"                 200
http_check "GET /api/status"             "http://127.0.0.1:9998/api/status"             200 onAir
http_check "GET /api/system"             "http://127.0.0.1:9998/api/system"             200 hostname
http_check "GET /api/network"            "http://127.0.0.1:9998/api/network"            200 interfaces
http_check "GET /api/config"             "http://127.0.0.1:9998/api/config"             200 ports
http_check "GET /api/health/stream"      "http://127.0.0.1:9998/api/health/stream"      200 overall

if [[ "$PROGRAM_READY" == "1" ]]; then
  section "Cloudflare tunnel (public ingress)"
  http_check "GET https://live.interdependent.dev/program/index.m3u8" \
             "https://live.interdependent.dev/program/index.m3u8" 200 EXTM3U

  section "Stream content reality check"
  # Final guard rail: actually pull a frame and confirm pixel content.
  # Black / solid-color frames are a usual symptom of an OBS scene state
  # bug that manifest-level checks miss.
  if command -v ffmpeg >/dev/null 2>&1; then
    if ffmpeg -hide_banner -loglevel error -y \
         -i "http://127.0.0.1:8888/program/index.m3u8" \
         -frames:v 1 -f image2 "$TMP.jpg" 2>/dev/null \
       && [[ -s "$TMP.jpg" ]]; then
      size=$(wc -c < "$TMP.jpg")
      if [[ $size -gt 2000 ]]; then
        ok "Live frame extracted  $(dim "(${size}B JPEG — has real content)")"
      else
        fail "Live frame is suspiciously tiny  $(dim "(${size}B — likely solid-color)")"
      fi
      rm -f "$TMP.jpg"
    else
      fail "ffmpeg could not extract a frame from /program"
    fi
  else
    echo "  $(dim "(ffmpeg not installed — skipping pixel-content check)")"
  fi
fi

# ─────────────────────────────────────────────────────────────────────
echo
total=$((PASS + FAIL))
if [[ $FAIL -eq 0 ]]; then
  echo "  $(green "PASS") $PASS/$total"
  exit 0
else
  echo "  $(red "FAIL") $FAIL of $total checks failed"
  exit 1
fi
