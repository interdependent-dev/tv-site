#!/usr/bin/env bash
# install.sh — download and unpack MediaMTX for this Mac Mini (darwin/arm64).
#
# Idempotent: re-running just verifies the binary and exits. Safe to call
# from test.sh as a prerequisite check.
set -euo pipefail

# Pin to a known-good release. Bump deliberately; don't auto-track "latest"
# so that a surprise upstream change can't break the broadcast mid-show.
MTX_VERSION="v1.17.1"

# Apple Silicon Mac Mini = darwin_arm64. If you ever move this to an Intel
# Mac, change to darwin_amd64; on Linux server, linux_amd64 / linux_arm64.
MTX_PLATFORM="darwin_arm64"

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${HERE}/mediamtx"
TARBALL="mediamtx_${MTX_VERSION}_${MTX_PLATFORM}.tar.gz"
URL="https://github.com/bluenviron/mediamtx/releases/download/${MTX_VERSION}/${TARBALL}"

# Short-circuit if the binary already exists and reports the expected version.
# MediaMTX prints its version on --help in the first line of stderr/stdout.
if [[ -x "${BIN}" ]] && "${BIN}" --version 2>/dev/null | grep -q "${MTX_VERSION#v}"; then
  echo "mediamtx ${MTX_VERSION} already installed at ${BIN}"
  exit 0
fi

echo "Downloading ${URL}"
curl -fL --retry 3 -o "${HERE}/${TARBALL}" "${URL}"

echo "Extracting…"
# The tarball contains the binary + a default mediamtx.yml (which we ignore —
# our own mediamtx.yml sits next to it and takes precedence when passed as arg).
tar -xzf "${HERE}/${TARBALL}" -C "${HERE}" mediamtx
rm "${HERE}/${TARBALL}"
chmod +x "${BIN}"

# Strip the macOS quarantine xattr so Gatekeeper doesn't block the first run.
# Without this, the first launch pops a "cannot be opened" dialog and the
# process exits. Harmless on files that have no quarantine attribute.
xattr -d com.apple.quarantine "${BIN}" 2>/dev/null || true

echo "Installed: $("${BIN}" --version)"
