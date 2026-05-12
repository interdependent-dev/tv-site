---
story_id: 1.4
story_key: 1-4-channelplayer-minimal-viable
epic: 1
sprint: Sprint 1 — Schedule and single-channel playback
title: <ChannelPlayer> minimal viable (hls.js + Safari native, deterministic seek)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: L
dependencies: ["1.2", "1.3", "0.2 (CORS confirmed)"]
blocks: ["1.5", "2.4", "3.2"]
risks: ["R2", "R3", "R10"]
project_context_refs:
  - "Cat 1 (hls.js >= 1.5, lowLatencyMode: false, two code paths)"
  - "Cat 2 (PlayerMode discriminated union, EpochMs/VideoSeconds branding)"
  - "Cat 3 (Player module contract)"
  - "Cat 7 #2 (schedule math sacred), #5 (autoplay muted), #12 (hls.js teardown), #13 (two code paths)"
pivot_doc_refs: ["§3 G1 (cold-load ≤2.5s)", "§7 (schedule math)", "§10.4 (player module)", "§11.4 (player chrome)"]
---

# Story 1.4: <ChannelPlayer> Minimal Viable

## User Story

As **a viewer landing on `interdependent.tv`**,
I want a live-feeling channel to start playing immediately at the correct computed position,
So that the FAST-channel illusion holds from frame one — and a second viewer on the same channel sees the same content within ±2s of me.

This is the acceptance demo for Sprint 1. Two browser windows on the same channel → ±2s in sync from cold load.

## Context

Three code-path rules that AI agents WILL get wrong without explicit guardrails:
- **hls.js path:** `MANIFEST_PARSED` → `currentTime = offsetSec` → `play()`. Order non-negotiable (Cat 7 #13).
- **Safari native path:** `loadedmetadata` → `currentTime = offsetSec` → `play()`. Distinct lifecycle event. Two separate code branches.
- **`hls.destroy()` before every reassign or unmount.** Strict-mode double-mount + channel-switch both leak otherwise (Cat 7 #12).

`Date.now()` is BANNED inside `currentProgram` (Cat 7 #2). Schedule math accepts `nowMs: EpochMs` as a parameter.

## Acceptance Criteria

### Component structure
1. Component at `app/(tv)/_components/ChannelPlayer/index.tsx`. Client component (`"use client"`).
2. Co-located: `usePlaybackPosition.ts` (pure function + hook wrapping it).
3. Co-located: `PlayerChrome.tsx` (overlay UI: logo, title, LIVE pill, time-remaining bar).

### Schedule math (pure, injectable)
4. `currentProgram(nowMs: EpochMs, channel: ChannelManifest): { asset, offsetSec: VideoSeconds, playlistIndex }` lives at `app/(tv)/_components/ChannelPlayer/usePlaybackPosition.ts` (or `lib/schedule.ts` for cross-component reuse).
5. **`Date.now()` is BANNED inside `currentProgram`.** It accepts `nowMs` parameter (Cat 7 #2).
6. Math: `positionInLoop = ((elapsedSec % L) + L) % L` — **double-mod** per pivot §7.2 (single-mod fails on negative skew / future anchors).
7. Returns `offsetSec` as branded `VideoSeconds`.

### Player bootstrap
8. On mount: read manifest (passed as prop or fetched from `/api/channels/{slug}/manifest`); get `nowMs` from `useServerTime`; compute `(currentAsset, offsetSec)`.
9. **Browser detection:** `if (videoEl.canPlayType('application/vnd.apple.mpegurl'))` → Safari native path; else → hls.js path (Cat 7 #13 — NOT user-agent sniffing).

### hls.js path
10. `import Hls from 'hls.js'`; `if (Hls.isSupported())` guard.
11. Create instance with `{ lowLatencyMode: false }` explicitly (Cat 1 — LL-HLS out of scope).
12. `hls.loadSource(currentAsset.hlsUrl)`.
13. **Wait for `Hls.Events.MANIFEST_PARSED`**, THEN set `video.currentTime = offsetSec`, THEN call `video.play()`.
14. **NEVER seek before parse. NEVER play before seek.** Order is the spec.

### Safari native path
15. `videoEl.src = currentAsset.hlsUrl`.
16. **Wait for `loadedmetadata` event**, THEN set `video.currentTime = offsetSec`, THEN call `video.play()`.

### Lifecycle
17. **`<video>` starts with `muted` attribute** (Cat 7 #5; pivot R10). Required for autoplay. Tap/click anywhere unmutes.
18. **On `ended`:** advance to `playlist[(currentIndex + 1) % playlist.length]`. Set `currentTime = 0`. Call `play()`. NO manifest re-fetch (it's in memory).
19. **On `error`** (network / manifest parse / segment 404): retry with exponential backoff (1s / 2s / 4s). After 3 failures, render `<PlayerError>` with "Try another channel" CTA.

### Cleanup (Cat 7 #12)
20. `useEffect` returns `() => hls?.destroy()` on unmount.
21. Channel switch: call `hls.destroy()` BEFORE reassigning the ref to a new instance.
22. No early-return guards that bypass teardown.

### Session writes
23. Writes session to LocalStorage via `useSession.updateSession({ lastChannelId, lastChannelSlug, lastAssetId, lastAssetHlsUrl, lastAssetTitle, lastPositionSec, exitedAtMs })`:
   - On `pause`.
   - On `timeupdate` (5s throttle — Cat 3 hook contract).
   - On `visibilitychange → hidden`.
   - On `beforeunload`.
   - **`lastAssetHlsUrl` and `lastAssetTitle`** are mandatory on every write per **D2-VOD-URL consensus (2026-05-12)**: session is the canonical VOD detour source. Pull both from the currently-playing playlist entry. Story 3.1 / 3.2 / 3.3 rely on these fields being present.

### Player chrome (pivot §11.4)
24. Top-left: channel logo + name.
25. Top-right: "LIVE" pill (red glow when in `live` mode; gray with "VOD" badge in `vod-detour` mode — Story 3.2 lights this up).
26. Bottom: now-playing title + thin time-remaining bar (resets at each program boundary; NO global scrubber for live mode).
27. Chrome fades after 3s of no mouse/touch; returns on movement.

### PlayerMode (Cat 2 + 7 #11)
28. `PlayerMode` discriminated union typed at `lib/types/player.ts`:
    ```ts
    type PlayerMode =
      | { kind: 'live'; channelSlug: string }
      | { kind: 'vod-detour'; assetId: string; resumeMs: number; returnToSlug: string }
      | { kind: 'returning'; channelSlug: string; countdownMs: number };
    ```
29. This story implements the `kind: 'live'` branch only. Stories 3.1 and 3.2 add the other two.

### Mux Data SDK — QoE telemetry instrumentation (Plan C, 2026-05-12 party consensus)

30. **Install `@mux/mux-data` (NOT `@mux/mux-video` — explicitly forbidden per project-context.md).** The Mux Data SDK is a viewer-side beacon — it instruments any HLS player and streams QoE events (startup time, rebuffer ratio, video startup failure, playback failure, video startup time, exit-before-video-start) to the Mux Data dashboard. **Delivery-agnostic: works on R2-served HLS without Mux Video as origin.**
31. **Initialize the SDK against the `<video>` element**, AFTER hls.js instance is created on non-Safari paths, AND on the native `<video>` directly on Safari. Pattern (hls.js path):
    ```ts
    import mux from '@mux/mux-data';
    mux.monitor(videoEl, {
      debug: false,
      hlsjs: hlsInstance,           // pass hls.js instance for hls.js-specific signals
      Hls: HlsCtor,                  // pass the Hls constructor too (Mux requirement)
      data: {
        env_key: process.env.NEXT_PUBLIC_MUX_DATA_KEY,
        viewer_user_id: session.sessionId,    // from useSession; anonymous UUID
        video_id: currentAsset.assetId,       // from manifest playlist
        video_title: currentAsset.title,
        video_series: channel.slug,           // channel as series; ambient/flagship taxonomy
        video_duration: currentAsset.durationSec * 1000, // ms per Mux Data spec
        player_name: 'interdependent-tv-channel-player',
        player_version: process.env.NEXT_PUBLIC_APP_VERSION ?? 'dev',
        player_init_time: Date.now(),         // OK to use Date.now() here per Cat 7 #2 — not schedule math
      },
    });
    ```
32. **`env_key`** read from `NEXT_PUBLIC_MUX_DATA_KEY` (publishable; client-side env var per Story 0.3 AC 17b). Build fails if missing.
33. **Asset transition** (loop advance → new asset at end-of-program): call `mux.emit(videoEl, 'videochange', { video_id, video_title, video_duration })` so Mux Data attributes events to the correct asset across loop boundaries.
34. **Channel switch:** call `mux.destroy(videoEl)` BEFORE `hls.destroy()` in the cleanup sequence (Cat 7 #12). Re-init on the new channel's player. Failing to destroy leaks beacon listeners and produces zombie session attribution.
35. **`viewer_user_id` privacy:** `session.sessionId` is a `crypto.randomUUID()` per Story 1.3 — no PII. Acceptable to ship to Mux Data per the §17.7 geo-aware-not-geo-fencing resolution.
36. **NOT a replacement for Story 3.4.** Mux Data captures DELIVERY QoE. Creator-economy metrics (completion rate, return-visitor rate per creator, cohort retention, channel loyalty) live in the custom `playback_events` pipeline (Story 3.4). Both layers ship; they answer different questions. Cross-referenced in Story 3.4.

### Mux Data acceptance tests

37. **Unit (jsdom):** mock `mux.monitor` → assert called with correct `env_key`, `viewer_user_id`, and `video_id` for a known fixture. Cat 4 — injected mock, no real Mux Data API.
38. **Playwright (browser-mode):** load a channel page; assert `window.mux` exists (the SDK loaded); assert at least one `mux.emit` call fired within 5s of play start (via mock interceptor or Mux Data debug mode).
39. **Manual:** Mux Data dashboard shows a session within 1 min of a real cold-load play. Captured as a screenshot in Story 1.5's perf-baseline report.

## Acceptance Tests

### Unit (jsdom, clock-injected — Cat 4)

- [ ] `currentProgram(1000n_ms, channel)` returns expected `(asset, offsetSec)` for known fixture.
- [ ] Double-mod: pass a `nowMs` BEFORE `epochAnchorMs` (negative elapsed); assert `positionInLoop` is correct (non-negative).
- [ ] Loop boundary: pass `nowMs` exactly at `epochAnchor + loopDurationSec * 1000`; assert returns asset 0 at offset 0.

### Playwright (browser-mode — Cat 4)

- [ ] **Seek sequence (hls.js):** assert `currentTime` is set AFTER `MANIFEST_PARSED` and BEFORE `play()`.
- [ ] **Seek sequence (Safari):** assert `currentTime` is set in `loadedmetadata` handler, BEFORE `play()`.
- [ ] **Memory leak:** Switch channels 10× in sequence; assert exactly 1 active `<video>`; assert 0 detached `HTMLMediaElement` instances (via DevTools heap).
- [ ] **Time sync (THE Sprint 1 demo gate):** open two browser contexts on `channel-1`; after 30s of playback, assert `video.currentTime` differs by ≤ 2s.
- [ ] **Refresh resilience:** refresh after 60s; assert resumes within ±2s of a parallel viewer.

### Smoke (manual / dev)

- [ ] Cold load `/` → channel autoplays at correct live position within 2.5s p95 (Story 1.5 measures formally).

## Dependencies

- **Upstream:** 1.2 (manifest endpoint), 1.3 (hooks), 0.2 (CORS confirmed for non-Safari segment fetch).
- **Downstream:** 1.5 (perf baseline), 2.4 (slug routing wraps this component), 3.2 (VOD detour extends `PlayerMode`).

## Risk Alignment (pivot §16)

| ID | Risk | Mitigation in this story |
|---|---|---|
| R2 | hls.js / Safari behavioral diffs | Two distinct code paths (AC 9–16). Story 4.3 expands the cross-browser test matrix. |
| R3 | Clock skew on viewer device | Schedule math consumes `useServerTime` (1.3); no `Date.now()` direct calls. |
| R10 | Browser autoplay policies | `muted` attribute on `<video>` (AC 17). Closed by pivot risk register. |
| R8 | Schedule edit during heavy viewing | Manifest 60s refresh (this story implements; mid-loop graceful switch handled here). Pivot has it Closed; this story enforces. |

## Implementation Notes

- **Branded `VideoSeconds` type** (Cat 2):
  ```ts
  export type VideoSeconds = number & { readonly __brand: 'VideoSeconds' };
  ```
  Conversion site: `currentProgram` returns `offsetSec` as `VideoSeconds`. The player's `video.currentTime = offsetSec` does the implicit cast to `number` at the DOM boundary.
- **Manifest background refresh (60s):** every 60s, refetch `/api/channels/{slug}/manifest`. Compare current asset's `assetId` vs. what the math NOW says should be playing. If different (schedule edit happened): tear down hls.js, reload with new asset, seek. Single retry. If still mismatched, render error overlay.
- **NO `Date.now()` outside the `useServerTime` provider.** Grep should return zero hits in `app/(tv)/_components/ChannelPlayer/**`.
- **`hls.js` import:** dynamic import is fine to keep initial JS bundle small. Pattern: `const Hls = (await import('hls.js')).default`. Trade-off: adds a microtask before `MANIFEST_PARSED` can fire; verify Story 1.5's 2.5s p95 budget.

## Definition of Done

- [ ] AC 1–39 met.
- [ ] All Playwright tests pass (including the ±2s time-sync gate — this is THE Sprint 1 demo gate G3).
- [ ] Memory-leak test green.
- [ ] R2/R3/R10 statuses updated in epic risk register.
- [ ] Mux Data session visible in Mux Data dashboard within 1 min of cold-load play (manual).
- [ ] Sprint-status.yaml `1-4-channelplayer-minimal-viable` → done.
