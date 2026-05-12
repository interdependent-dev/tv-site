---
story_id: 0.2
story_key: 0-2-name-hls-encoding-pipeline-owner
epic: 0
sprint: Pre-Sprint-1 Prep
title: Encoding pipeline + R2 storage setup (Chris-owned)
status: ready-for-dev
priority: Must (blocker for Sprint 1 Story 1.4 demo)
owners:
  - Chris (end-to-end per D13)
effort: M
dependencies: ["0.1 (need first asset confirmed)"]
blocks: ["1.1", "1.4"]
risks: ["R1"]
project_context_refs: ["Cat 1 (R2 + Edge runtime contract)", "Cat 7 #8 (CORS)", "Cat 7 #13 (player code paths)"]
pivot_doc_refs: ["ADR-003", "§8.1 assets table", "§14.1 scale path"]
decisions_resolved: ["D12", "D13", "D14", "D15", "D16", "D17", "D18", "D19"]
---

# Story 0.2: Encoding Pipeline + R2 Storage Setup

## User Story

As **Chris (encoding pipeline owner per D13)**,
I want a reproducible FFmpeg encoding pipeline + two configured R2 buckets,
So that Story 1.1's seed has a working `hls_url` and Story 1.4's player has CORS-clean HLS segments to consume from the browser.

## Context

Pivot ADR-003: R2 + FFmpeg locally for POC; migrate to Cloudflare Stream at scale. The 2026-05-12 MVP triage settled on this baseline (D12). Ownership shifted from Vik to Chris (D13) — Chris owns the encoding pipeline end-to-end.

Two surfaces are involved:
1. **`tv-masters`** — private R2 bucket, source-of-truth for the MP4/MOV originals (D18).
2. **HLS bucket** — public-via-custom-domain `cdn.interdependent.tv` for HLS manifests + segments. Browser fetches segments DIRECTLY from this domain (project-context.md Cat 7 #8 — not proxied through Workers).

The architectural rule (project-context.md Cat 1) is non-negotiable: browser → R2 direct fetch for segments. CORS must be configured on the HLS bucket before any non-Safari browser test (Story 1.4 demo). Safari is permissive → testing only on Safari produces false positives.

## Decisions Already Resolved (2026-05-12 party-mode triage)

| ID | Decision | Resolution |
|---|---|---|
| D12 | Encoding model | In-house FFmpeg (per ADR-003) |
| D13 | Encoding owner | Chris (end-to-end) |
| D14 | URL structure | HLS: `cdn.interdependent.tv/<asset-uuid>/master.m3u8` (subdomain); viewer URLs: `interdependent.tv/channel-N` (apex, handled in Story 2.4) |
| D15 | ABR ladder | 1080p / 720p / 480p, H.264 baseline, AAC 128k stereo, 6s segments |
| D16 | Deadline | 1 channel HLS in R2 by Sprint 1 day 5; remaining 4 by Sprint 2 day 5 |
| D17 | CORS | `Access-Control-Allow-Origin: *` for POC; explicit acceptance test below |
| D18 | Master storage | `tv-masters` R2 bucket; path `masters/{asset_id}/source.{ext}` |
| D19 | Audit trail | Deferred (post-POC) |

## Acceptance Criteria

1. **Two R2 buckets provisioned:**
   - `tv-masters` (private, service-token access) — source MP4/MOV uploaded here.
   - HLS bucket (e.g., `tv-hls`) — public read; served via custom domain `cdn.interdependent.tv`.
2. **Custom domain `cdn.interdependent.tv` resolves with valid SSL.** A `curl -I https://cdn.interdependent.tv/<sample-asset-uuid>/master.m3u8` returns `HTTP 200` once content is uploaded.
3. **FFmpeg encode script lives in the repo** at `scripts/encode-hls.sh` (or similar). Idempotent; takes an input MP4/MOV path + asset UUID; outputs the 3-rung HLS ladder per D15 to a local directory.
4. **R2 upload script lives in the repo** at `scripts/upload-hls.sh` (or similar). Takes the local HLS output directory + asset UUID; uploads to the HLS bucket under `{asset-uuid}/` prefix. Uses `wrangler r2 object put` or `rclone` with R2 credentials.
5. **First-asset encode complete by Sprint 1 day 5.** Channel 1's first asset (per Story 0.1) is encoded + uploaded; `master.m3u8` is reachable via `cdn.interdependent.tv/{asset-uuid}/master.m3u8`.
6. **CORS acceptance test (D17) passes:**
   - From DevTools console on `https://example.com` (any non-same-origin context), run `fetch('https://cdn.interdependent.tv/<asset-uuid>/master.m3u8')`.
   - Response succeeds; no CORS error in console.
   - Documented as a one-step manual test in this story's close-out notes.
7. **Cloudflare R2 service token** is created with restricted scope (read+write on the two buckets only) and stored in `.dev.vars` (gitignored). Production tokens stored via `wrangler secret put`.
8. **Encoding documentation** added as a one-page README at `scripts/README.md` explaining: how to encode, how to upload, where masters live, where HLS lives, CORS expectations.

## Acceptance Tests / Verification

**Manual verification checklist:**

- [ ] `wrangler r2 bucket list` shows `tv-masters` and `tv-hls` (or chosen names).
- [ ] DNS resolves `cdn.interdependent.tv` → R2 custom-domain endpoint.
- [ ] SSL cert valid for `cdn.interdependent.tv` (no browser warnings).
- [ ] First asset's `master.m3u8` returns 200 from `curl`.
- [ ] First asset's `master.m3u8` plays in a one-page HTML test harness with hls.js + Safari (both).
- [ ] CORS preflight from cross-origin page succeeds.
- [ ] `scripts/encode-hls.sh` is runnable and reproducible.
- [ ] `.dev.vars` is in `.gitignore`.

**Automated acceptance (carry into Story 1.4 demo):**

- [ ] Channel 1's `hls_url` in the seeded `assets` row is non-null and reachable.

## Dependencies

- **Upstream:** Story 0.1 (need Channel 1's first asset confirmed for the first encode).
- **Downstream (blocks):**
  - Story 1.1 (Data model + seed) — needs a non-null `hls_url` to seed.
  - Story 1.4 (`<ChannelPlayer>` minimal viable) — needs a CORS-clean HLS endpoint for the demo.

## Risk Alignment (pivot §16)

| ID | Risk | Status this story closes | Mitigation |
|---|---|---|---|
| R1 | Asset encoding pipeline absent | Closed (when AC 1–6 met) | This story IS the closure. The pipeline is named, scripted, and reproducible. |
| R10 | Browser autoplay policies block silent autoplay | Closed (cross-ref) | `muted` attribute on `<video>` is Story 1.4 responsibility; this story enables it by ensuring HLS plays at all. |
| R2 | hls.js / Safari behavioral diffs | Open until Sprint 4 (4.3) | CORS-clean delivery is the precondition; behavioral testing is Story 4.3. |

## Implementation Notes

**Recommended FFmpeg command shape** (one-asset, 3-rung ladder, 6s segments):

```bash
# Encode 1080p
ffmpeg -i "$INPUT" -c:v libx264 -profile:v baseline -level 4.0 -preset veryfast \
  -b:v 4500k -maxrate 5000k -bufsize 9000k -vf "scale=-2:1080" \
  -c:a aac -ar 48000 -b:a 128k -ac 2 \
  -hls_time 6 -hls_playlist_type vod -hls_segment_filename "out/1080p/segment_%03d.ts" \
  -f hls "out/1080p/index.m3u8"

# Same shape for 720p (-b:v 2500k -vf "scale=-2:720") and 480p (-b:v 1200k -vf "scale=-2:480")

# Generate master.m3u8 manually or via a wrapper
```

**CORS config for the HLS bucket** (Cloudflare dashboard → R2 → bucket → Settings → CORS):

```json
[
  {
    "AllowedOrigins": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedHeaders": ["*"],
    "MaxAgeSeconds": 86400
  }
]
```

POC permissiveness is acceptable per D17. Lock to `https://interdependent.tv` + preview-branch origins post-launch.

**Cache headers on segments** (pivot §12.2):
- `Cache-Control: public, max-age=604800` (7 days) on HLS segments — these are immutable once encoded.
- `Cache-Control: public, max-age=60` on `master.m3u8` — allows future schedule edits without aggressive caching.

## Definition of Done

- [ ] AC 1–8 met.
- [ ] CORS acceptance test passes in a non-Safari browser.
- [ ] R1 status in epic file risk register = Closed.
- [ ] Sprint-status.yaml story `0-2-name-hls-encoding-pipeline-owner` → done.
- [ ] Story 1.1 and Story 1.4 unblocked.
