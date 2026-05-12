---
story_id: 3.2
story_key: 3-2-vod-detour-mode-in-player
epic: 3
sprint: Sprint 3 — Resume / live decision UX
title: VOD detour mode in player (PlayerMode 'vod-detour' + 'returning')
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: M
dependencies: ["3.1"]
blocks: []
risks: []
project_context_refs:
  - "Cat 2 (PlayerMode discriminated union)"
  - "Cat 7 #11 (VOD detour pattern, ADR-006)"
pivot_doc_refs: ["§11.4 (player chrome — VOD badge)", "ADR-006", "Sprint 3 Story 3.2"]
---

# Story 3.2: VOD Detour Mode in Player

## User Story

As **a viewer who chose Continue from the Resume Modal**,
I want my last-watched asset to play standalone from where I left off, with a full scrubber, then auto-return to the live channel when it ends,
So that "coming back" actually lets me finish what I started — without trapping me out of live TV.

## Context

ADR-006: "Continue" plays the asset standalone as VOD with `currentTime = lastPositionSec`. When the asset `ended`s, navigate to `/[returnToSlug]` and resume live. Player chrome shows "VOD — returning to *Channel* live in M:SS" once <30s remain.

PlayerMode union (Cat 2 + 7 #11), updated per D2-VOD-URL consensus:
```ts
type PlayerMode =
  | { kind: 'live'; channelSlug: string }
  | { kind: 'vod-detour'; assetId: string; hlsUrl: string; title: string; resumeMs: number; returnToSlug: string }
  | { kind: 'returning'; channelSlug: string; countdownMs: number };
```
The `hlsUrl` and `title` are part of the VOD detour mode payload — passed in by the Resume Modal from `session.lastAssetHlsUrl` / `session.lastAssetTitle`. No separate fetch.

Story 1.4 implemented `kind: 'live'` only. This story adds `vod-detour` and `returning`.

## Acceptance Criteria

### Mode plumbing

1. `<ChannelPlayer>` (extended from Story 1.4) accepts a `mode: PlayerMode` prop OR reads from session / query params.
2. Mode transitions:
   - `live → vod-detour`: triggered by Resume Modal's Continue button (Story 3.1 Rows 2/3/4).
   - `vod-detour → returning`: triggered when <30s remain in the VOD asset.
   - `returning → live`: triggered on VOD `ended` event → `router.push('/{returnToSlug}')` → fresh page load resumes live.

### VOD detour behavior

3. **Player accepts `{ mode: { kind: 'vod-detour', assetId, hlsUrl, title, resumeMs, returnToSlug } }`.** Per **D2-VOD-URL consensus (2026-05-12 party)**, the `hlsUrl` and `title` are passed in by the Resume Modal — sourced from `session.lastAssetHlsUrl` / `session.lastAssetTitle`. The player does NOT fetch a manifest to resolve the asset; the URL is already canonical from session.
4. **No `/api/assets/{id}` endpoint exists in POC.** Session is the only VOD-detour source. If the modal couldn't populate `hlsUrl` (legacy session), it never reaches VOD detour — it degrades to row 5 in Story 3.1.
5. Loads HLS using `hlsUrl` from mode prop; on `MANIFEST_PARSED` (hls.js) / `loadedmetadata` (Safari), sets `currentTime = resumeMs / 1000` (convert ms→seconds at the conversion boundary). Plays.
6. **Full scrubber visible** in player chrome (`<input type="range">` or custom). User can seek freely within the asset. (Live mode has no scrubber — pivot §11.4.)
7. **`<video>` chrome shows:**
   - Top-right pill: "VOD" (gray) instead of "LIVE" (red).
   - Below pill: "← returning to *Channel Name*" link (allows early exit).
8. **Countdown trigger:** when `video.duration - video.currentTime <= 30`, mode transitions to `returning`. Chrome updates to show "Returning to *Channel* live in M:SS" badge with M:SS = remaining time, ticking down each second.

### Returning behavior

9. On VOD asset `ended` event (or on early-exit click): `router.push('/{returnToSlug}')`. This triggers a fresh navigation; the destination's `<ChannelPlayer>` mounts in `kind: 'live'` mode.
10. **Early exit "Go live now" CTA:** clickable text or button in chrome that immediately `router.push('/{returnToSlug}')` regardless of remaining time.
11. **Asset-near-end + scrubber-forward edge case (John's §17 party flag):** if user scrubs forward such that remaining time crosses below 30s threshold, the badge transitions in without flickering. If user scrubs backward past 30s, badge hides (transition `returning → vod-detour`).

### URL structure

12. **VOD detour does NOT have a dedicated URL.** It lives on the path it was invoked from (typically `/`). Refresh during VOD detour returns to the Resume Modal flow (which re-decides based on session state).
13. Alternative considered + rejected: dedicated `/vod/[assetId]` route would let viewers share VOD detours, but Pivot non-goal §2.4 #2 says no DRM. VOD URLs that don't carry rights metadata would conflict with future DRM. Defer the dedicated VOD route to post-POC.

### Analytics events

14. On entering `vod-detour`: fire `vod_detour_enter` event with `{ assetId, resumeMs }` (Story 3.4 wires the endpoint; this story emits the call).
15. On exiting (asset ended OR early exit): fire `vod_detour_exit` with `{ assetId, exitMode: 'ended' | 'manual' }`.

## Acceptance Tests

**Playwright:**
- [ ] Click Resume Modal Continue (Row 3 scenario): VOD detour starts at correct `resumeMs`; full scrubber visible.
- [ ] Asset plays to end → `router.push('/{returnToSlug}')` fires → live channel renders.
- [ ] At T-30s, badge appears with "Returning to *Channel* live in 0:30" countdown; ticks down.
- [ ] Scrub forward past T-30s threshold: badge transitions in without flicker.
- [ ] Scrub backward past T-30s threshold: badge hides.
- [ ] Early exit click: navigates to live immediately.

**Unit (jsdom):**
- [ ] PlayerMode transition function (pure): given `(mode, event)` returns the next mode correctly across all 6 transition cases.

## Dependencies

- **Upstream:** 3.1 (Resume Modal triggers the mode).
- **Downstream:** 3.4 (analytics events get wired to API).

## Risk Alignment

(No direct §16 risks; UX correctness is the spec.)

## Implementation Notes

- **Asset URL source (RESOLVED per D2-VOD-URL party consensus, 2026-05-12):** Session is canonical. `useSession` exposes `session.lastAssetHlsUrl` and `session.lastAssetTitle` (added by Story 3.1 per D3 additive-evolution). Story 1.4 and 2.4's exit-write `updateSession` call sites populate both fields on every session save. Resume Modal reads them and passes them as PlayerMode props. No manifest indirection, no `/api/assets/{id}` endpoint in POC. If the session predates the field (e.g., legacy LocalStorage from a pre-Sprint-3 deploy), the modal degrades to row 5 — no crash, no fetch attempt.
- **PlayerMode transition function:**
  ```ts
  function nextMode(mode: PlayerMode, event: PlayerEvent): PlayerMode {
    switch (event.type) {
      case 'continue-vod': return { kind: 'vod-detour', ...event };
      case 'enter-returning-window': return { kind: 'returning', ... };
      case 'exit-returning-window': return { kind: 'vod-detour', ... };
      case 'asset-ended': return { kind: 'live', channelSlug: mode.returnToSlug }; // via router.push
      case 'early-exit': return { kind: 'live', channelSlug: mode.returnToSlug };
    }
  }
  ```
  Pure. Unit-testable. Discriminated.
- **Scrubber implementation:** standard `<input type="range" min="0" max="{duration}" value="{currentTime}" onChange={...} />`. Two-way bind to `video.currentTime`. Throttle the change handler at 50ms to avoid flooding.

## Definition of Done

- [ ] AC 1–15 met.
- [ ] Playwright tests pass for all transition paths + edge cases.
- [ ] Sprint-status.yaml `3-2-vod-detour-mode-in-player` → done.
