---
story_id: 2.3
story_key: 2-3-channelguide-quick-switch-row
epic: 2
sprint: Sprint 2 — Multi-channel and channel guide
title: <ChannelGuide> quick-switch row (horizontal scrollable tiles)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl) + Sally (UX review)"]
effort: L
dependencies: ["2.2"]
blocks: ["2.4"]
risks: []
project_context_refs:
  - "Cat 3 (component organization, _components/ChannelGuide/)"
  - "Cat 7 #14 (eventually consistent ~60s; guide refresh cadence)"
pivot_doc_refs: ["§10.5 (Guide module)", "§11.3 (channel guide UX)", "§3 G2 (switch latency ≤500ms)"]
---

# Story 2.3: <ChannelGuide> Quick-Switch Row

## User Story

As **a viewer who wants to change channels**,
I want a horizontal scrollable row of channel tiles showing what's currently playing,
So that I can pick a channel at a glance and switch with one click — within 500ms (G2).

## Context

The Guide is the second-most-load-bearing UI surface after the Player. It's the "channel changer" of the FAST experience. Pivot §11.3 describes two view modes: **Quick switch (default)** = horizontal row of tiles; **EPG (toggle)** = full-screen grid with time slots. This story implements quick switch only. EPG grid is out of scope for POC (DEFERRED per pivot §11.3 + non-goal §2.4 #8).

## Acceptance Criteria

1. Component at `app/(tv)/_components/ChannelGuide/index.tsx`. Client component.
2. Co-located: `ChannelRow.tsx` (the row), `ChannelTile.tsx` (the individual tile).
3. **Data source:** fetches from `/api/channels` on mount; refreshes every 30s via `setInterval` (matching the KV TTL — pivot §12.1).
4. **Layout:**
   - Horizontal scrollable row of channel tiles.
   - Mobile: thumb-reach optimized; tile width ~ 60% viewport width; horizontal scroll-snap.
   - Desktop: 5 tiles visible without scrolling (5 channels in POC); hover effect (subtle scale or border).
5. **Each tile renders:**
   - Channel logo (top-left).
   - Current program poster as background (or solid color fallback).
   - Title at bottom.
   - "LIVE" pill.
   - Time-remaining bar (fills based on `current.endsAtMs - serverNow / current.endsAtMs - current.startsAtMs`).
6. **Active channel** (current `/[slug]` route's channel) is visually distinct (highlighted border or background).
7. **Click on tile** triggers Next.js `router.push('/${slug}')` (D14 — apex routing, no `/watch/` prefix). NOT a full page reload.
8. **Switch latency:** from click to player swap visible, ≤ 500ms (pivot G2). Verified in 2.5 soak test.
9. **Keyboard nav (per pivot §11.6, MVP subset):**
   - `↑↓` cycles channel selection.
   - `Enter` activates highlighted channel.
   - `1–5` jumps to channel by `sort_order + 1` (TV-style channel numbers).
10. **Open/close UX:**
    - Mobile: bottom-edge swipe up reveals; swipe down dismisses.
    - Desktop: `G` or `↓` key opens; `Esc` closes; on-screen button toggle.
    - Idle: closes after 5s of no interaction once revealed (configurable).

## Acceptance Tests

**Playwright (browser-mode):**
- [ ] On `/channel-1`, opening the guide and clicking `channel-3` tile navigates to `/channel-3` without full page reload.
- [ ] Tile shows the correct current program from `/api/channels`.
- [ ] Active-channel highlight tracks the URL.
- [ ] Time-remaining bar advances over a 30s observation window.
- [ ] Keyboard: pressing `3` while on `/channel-1` navigates to `/channel-3` (assuming channel-3's sort_order is 2).

**Unit (jsdom):**
- [ ] Tile renders given a mock `channel` prop.
- [ ] Renders nothing (skeleton) while `useEffect` initial fetch is pending.

## Dependencies

- **Upstream:** 2.2 (channels list endpoint).
- **Downstream:** 2.4 (slug routing).

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| (no direct §16 risk) | Switch latency G2 = ≤500ms p95 | This story's design (router.push, hls.js destroyed cleanly, manifest fetch via KV hit) is the path to meeting G2. Verified in 2.5. |

## Implementation Notes

- **Posters:** Pivot Sprint 2 Story 2.3 calls for "current program poster as background." For POC, an `<img>` with `object-fit: cover` is fine. Lazy loading via `loading="lazy"` for non-visible tiles.
- **30s refresh interval:** matches `/api/channels` KV TTL. Avoid sub-30s polling — wastes Workers requests.
- **`<ChannelTile>` is pure render given channel + serverNow.** Test in isolation. The time-remaining computation is deterministic given inputs.
- **Active state:** read `usePathname()` from `next/navigation`; compare with each channel's slug.
- **Keyboard handler:** scope to the document (`document.addEventListener('keydown', ...)`) with proper cleanup. Skip handling when an input/textarea has focus (`document.activeElement.tagName === 'INPUT'`).
- **Mobile swipe:** use the Pointer Events API (project-context.md preference for native over libraries). A small custom handler is fewer KB than `react-swipeable`.

## Definition of Done

- [ ] AC 1–10 met.
- [ ] Playwright tests pass.
- [ ] Sprint-status.yaml `2-3-channelguide-quick-switch-row` → done.
