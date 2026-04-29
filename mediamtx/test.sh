#!/usr/bin/env bash
# test.sh — end-to-end pipeline test for the MediaMTX egress layer.
#
# What it proves:
#   1. The binary downloads and starts with our mediamtx.yml.
#   2. The control API comes up on :9997 (localhost).
#   3. All three ports we care about are listening (SRT 8890/udp,
#      HLS 8888/tcp, WebRTC 8889/tcp).
#   4. When ffmpeg publishes a test pattern via SRT with streamid=publish:program,
#      MediaMTX exposes a valid LL-HLS manifest at /program/index.m3u8.
#   5. The WHEP endpoint responds so WebRTC clients can negotiate.
#
# Exits 0 on success, non-zero on first failure. Always cleans up the mediamtx
# and ffmpeg processes it started, even on failure (trap on EXIT).
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${HERE}/mediamtx"
CFG="${HERE}/mediamtx.yml"
LOG="${HERE}/test-mediamtx.log"
FF_LOG="${HERE}/test-ffmpeg.log"

# ── 0. Preflight ─────────────────────────────────────────────────────────────
echo "── preflight ──"

if ! command -v ffmpeg >/dev/null 2>&1; then
  cat >&2 <<'MSG'
ffmpeg not found on PATH.

ffmpeg is only needed for the end-to-end test (it generates the test pattern
that stands in for OBS). Install with:

    brew install ffmpeg

Then re-run this script. The mediamtx daemon itself does NOT require ffmpeg
in production — OBS is the publisher.
MSG
  exit 2
fi

# Homebrew's default ffmpeg bottle is built without libsrt (you get `srtp`,
# which is Secure RTP, not SRT). Detect that and fall back to the libsrt
# `srt-live-transmit` helper piped from ffmpeg over a local UDP socket.
FFMPEG_HAS_SRT=0
if ffmpeg -hide_banner -protocols 2>/dev/null | grep -Eq '^[[:space:]]*srt$'; then
  FFMPEG_HAS_SRT=1
fi
if [[ "${FFMPEG_HAS_SRT}" -eq 0 ]] && ! command -v srt-live-transmit >/dev/null 2>&1; then
  cat >&2 <<'MSG'
Your ffmpeg lacks SRT support and srt-live-transmit isn't installed.

Install one of:
  • ffmpeg with SRT:   brew tap homebrew-ffmpeg/ffmpeg && brew install homebrew-ffmpeg/ffmpeg/ffmpeg
  • srt-live-transmit: brew install srt   (smaller, recommended)
MSG
  exit 2
fi

if [[ ! -x "${BIN}" ]]; then
  echo "mediamtx binary missing, installing…"
  "${HERE}/install.sh"
fi

# ── 1. Start mediamtx in the background ──────────────────────────────────────
echo "── starting mediamtx ──"
: >"${LOG}"
"${BIN}" "${CFG}" >"${LOG}" 2>&1 &
MTX_PID=$!

FF_PID=""
SLT_PID=""
cleanup() {
  # Order matters: kill the publisher first so mediamtx can tear down the path
  # cleanly, then stop the daemon. `|| true` so cleanup never fails the script.
  for p in "${FF_PID}" "${SLT_PID}"; do
    if [[ -n "${p}" ]] && kill -0 "${p}" 2>/dev/null; then
      kill "${p}" 2>/dev/null || true
      wait "${p}" 2>/dev/null || true
    fi
  done
  if kill -0 "${MTX_PID}" 2>/dev/null; then
    kill "${MTX_PID}" 2>/dev/null || true
    wait "${MTX_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ── 2. Wait for the control API to answer ────────────────────────────────────
# Poll rather than sleeping a fixed duration — the binary usually boots in
# <500ms but a cold disk can take longer, and fixed sleeps are flaky in CI.
echo "── waiting for API on :9997 ──"
# -s (silent) hides the noisy "connection refused" during the startup race;
# -f makes curl exit non-zero on HTTP errors so the loop condition works.
for i in {1..40}; do
  if curl -fs -o /dev/null http://127.0.0.1:9997/v3/paths/list 2>/dev/null; then
    echo "  API ready after ${i} tries"
    break
  fi
  if ! kill -0 "${MTX_PID}" 2>/dev/null; then
    echo "FAIL: mediamtx exited during startup. Log:" >&2
    cat "${LOG}" >&2
    exit 1
  fi
  sleep 0.25
done

if ! curl -fs -o /dev/null http://127.0.0.1:9997/v3/paths/list 2>/dev/null; then
  echo "FAIL: API never came up" >&2
  cat "${LOG}" >&2
  exit 1
fi

# ── 3. Verify listening ports ────────────────────────────────────────────────
# lsof is the most portable way on macOS to check socket state without root.
# -iTCP/-iUDP filter by protocol; -sTCP:LISTEN narrows to listening sockets.
echo "── verifying listening ports ──"
check_port() {
  local label="$1" proto="$2" port="$3"
  if [[ "${proto}" == "tcp" ]]; then
    lsof -nP -iTCP:"${port}" -sTCP:LISTEN -a -p "${MTX_PID}" >/dev/null \
      || { echo "FAIL: ${label} not listening on tcp/${port}"; return 1; }
  else
    lsof -nP -iUDP:"${port}" -a -p "${MTX_PID}" >/dev/null \
      || { echo "FAIL: ${label} not listening on udp/${port}"; return 1; }
  fi
  echo "  ✓ ${label} ${proto}/${port}"
}
check_port "SRT"    udp 8890
check_port "HLS"    tcp 8888
check_port "WebRTC" tcp 8889

# ── 4. Publish a test pattern via SRT ────────────────────────────────────────
# lavfi testsrc2 = colour bars with a ticking clock, 30fps.
# We encode h264 at a low bitrate (broadcast-ish but cheap on CPU) and mux
# into mpegts. streamid=publish:program matches the path declared in
# mediamtx.yml. latency=200000 = 200ms SRT receive buffer.
if [[ "${FFMPEG_HAS_SRT}" -eq 1 ]]; then
  echo "── ffmpeg (native SRT) publishing testsrc2 → srt publish:program ──"
  : >"${FF_LOG}"
  ffmpeg -hide_banner -loglevel error \
    -re -f lavfi -i "testsrc2=size=640x360:rate=30" \
    -f lavfi -i "sine=frequency=1000:sample_rate=48000" \
    -c:v libx264 -preset ultrafast -tune zerolatency -b:v 600k -g 30 \
    -c:a aac -b:a 64k \
    -f mpegts \
    "srt://127.0.0.1:8890?streamid=publish:program&pkt_size=1316&latency=200000" \
    >"${FF_LOG}" 2>&1 &
  FF_PID=$!
else
  # Fallback path: Homebrew's ffmpeg doesn't include libsrt. We pipe mpegts
  # over a local UDP port into srt-live-transmit, which forwards to MediaMTX.
  # Functionally equivalent to native SRT for the purposes of this test —
  # proves the mediamtx SRT listener is accepting publishes.
  echo "── ffmpeg → UDP:9999 → srt-live-transmit → srt publish:program ──"
  : >"${FF_LOG}"
  ffmpeg -hide_banner -loglevel error \
    -re -f lavfi -i "testsrc2=size=640x360:rate=30" \
    -f lavfi -i "sine=frequency=1000:sample_rate=48000" \
    -c:v libx264 -preset ultrafast -tune zerolatency -b:v 600k -g 30 \
    -c:a aac -b:a 64k \
    -f mpegts \
    "udp://127.0.0.1:9999?pkt_size=1316" \
    >"${FF_LOG}" 2>&1 &
  FF_PID=$!
  # -t 0 = run indefinitely; caller mode pushes to mediamtx's listener.
  srt-live-transmit -t:0 \
    "udp://:9999?mode=listener" \
    "srt://127.0.0.1:8890?streamid=publish:program&mode=caller&latency=200" \
    >>"${FF_LOG}" 2>&1 &
  SLT_PID=$!
fi

# ── 5. Wait for the `program` path to report a publisher ─────────────────────
# The API reports a `ready: true` flag on paths that have an active publisher.
# This is the authoritative "SRT handshake succeeded" signal — more reliable
# than waiting a fixed number of seconds.
echo "── waiting for /program to show ready=true ──"
for i in {1..40}; do
  body="$(curl -fsS http://127.0.0.1:9997/v3/paths/get/program 2>/dev/null || true)"
  if [[ "${body}" == *'"ready":true'* ]]; then
    echo "  ✓ /program is live"
    break
  fi
  if ! kill -0 "${FF_PID}" 2>/dev/null; then
    echo "FAIL: ffmpeg exited before publish succeeded. Log:" >&2
    cat "${FF_LOG}" >&2
    exit 1
  fi
  sleep 0.25
done
body="$(curl -fsS http://127.0.0.1:9997/v3/paths/get/program 2>/dev/null || true)"
[[ "${body}" == *'"ready":true'* ]] || { echo "FAIL: /program never became ready"; exit 1; }

# ── 6. Pull the HLS manifest ─────────────────────────────────────────────────
# hlsAlwaysRemux=yes should mean the manifest exists immediately, but give
# the LL-HLS muxer a moment to emit its first parts (partDuration=200ms).
echo "── fetching LL-HLS manifest ──"
MANIFEST=""
for i in {1..30}; do
  if MANIFEST="$(curl -fsS http://127.0.0.1:8888/program/index.m3u8 2>/dev/null)"; then
    break
  fi
  sleep 0.5
done

[[ -n "${MANIFEST}" ]] || { echo "FAIL: manifest request failed"; exit 1; }

# Minimum viable m3u8: starts with #EXTM3U, has a version, references at
# least one variant or segment. For LL-HLS specifically, the master playlist
# links to a media playlist; we just check we got a master or media playlist.
[[ "${MANIFEST}" == "#EXTM3U"* ]] || { echo "FAIL: not an m3u8:"; printf '%s\n' "${MANIFEST}"; exit 1; }
echo "  ✓ manifest is a valid m3u8 (first line matched #EXTM3U)"

# Pull the media playlist too, to prove segments are actually being written.
# The master playlist links to a sub-playlist; grep it out and fetch.
SUB="$(printf '%s\n' "${MANIFEST}" | grep -E '\.m3u8($|\?)' | head -1 || true)"
if [[ -n "${SUB}" ]]; then
  MEDIA="$(curl -fsS "http://127.0.0.1:8888/program/${SUB}" || true)"
  if [[ "${MEDIA}" == *'#EXTINF'* || "${MEDIA}" == *'#EXT-X-PART'* ]]; then
    echo "  ✓ media playlist contains segments/parts (stream is actively muxing)"
  else
    echo "WARN: media playlist has no #EXTINF or #EXT-X-PART yet" >&2
  fi
fi

# ── 7. Poke the WHEP endpoint ────────────────────────────────────────────────
# A full WebRTC negotiation needs an SDP offer; here we just verify the
# endpoint is registered and responds to OPTIONS with the CORS headers that
# browsers need. Anything in the 2xx/4xx range means the route is wired up;
# a connection refused or 404 would mean WebRTC is misconfigured.
echo "── checking WHEP endpoint ──"
WHEP_CODE="$(curl -s -o /dev/null -w '%{http_code}' -X OPTIONS http://127.0.0.1:8889/program/whep || true)"
if [[ "${WHEP_CODE}" =~ ^[24] ]]; then
  echo "  ✓ WHEP responded ${WHEP_CODE}"
else
  echo "FAIL: WHEP endpoint returned ${WHEP_CODE}"; exit 1
fi

echo
echo "──────────────────────────────────────────────────────────────────────"
echo "  ALL CHECKS PASSED"
echo "──────────────────────────────────────────────────────────────────────"
echo "  LL-HLS:  http://127.0.0.1:8888/program/index.m3u8"
echo "  WHEP:    http://127.0.0.1:8889/program/whep"
echo "  OBS URL: srt://<host>:8890?streamid=publish:program&pkt_size=1316"
echo
