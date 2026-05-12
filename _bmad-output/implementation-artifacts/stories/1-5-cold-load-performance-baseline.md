---
story_id: 1.5
story_key: 1-5-cold-load-performance-baseline
epic: 1
sprint: Sprint 1 — Schedule and single-channel playback
title: Cold-load performance baseline (p95 ≤ 2.5s to first frame)
status: ready-for-dev
priority: Should
owners: ["Amelia (impl)"]
effort: S
dependencies: ["1.4"]
blocks: []
risks: []
project_context_refs:
  - "Cat 1 (Edge runtime cold-start considerations)"
  - "Cat 4 (perf testing via Lighthouse + Playwright trace)"
pivot_doc_refs: ["§3 G1 (cold load ≤2.5s p95)", "Sprint 1 Story 1.5"]
---

# Story 1.5: Cold-Load Performance Baseline

## User Story

As **the platform**,
I want a measured cold-load performance baseline against the G1 acceptance gate (≤2.5s p95 time-to-first-frame on broadband),
So that Sprint 1 exit is provable, not a guess.

## Context

This is the first acceptance-gate verification in the epic. Cold load = no browser cache, no warm DNS, no warm KV. Time-to-first-frame = the moment `<video>` actually paints a frame, not just the `play()` call returning.

Two parallel fetches must NOT waterfall (pivot Story 1.5 AC):
1. `/api/time` + `/api/channels/{slug}/manifest` (Workers / KV).
2. `hls.js` bundle download (npm CDN or inlined).

## Acceptance Criteria

1. **Lighthouse run** against a deployed Workers preview env (NOT localhost — Cat 4). Simulated broadband (Cable / 4G — pick one and document).
2. **Time-to-First-Contentful-Paint (FCP)** captured.
3. **Time-to-First-Video-Frame** measured via Playwright trace: hook into `<video>` element's first `timeupdate` event after `play()` resolves; record elapsed since navigation start.
4. **p95 of 10 cold-load runs ≤ 2.5s** for first-video-frame.
5. **No waterfall** between `/api/channels/{slug}/manifest` fetch and hls.js bundle load. Verified via DevTools Network panel (or Playwright HAR export):
   - Both requests start within 100ms of each other.
   - Neither waits on the other's response before starting.
6. **Baseline report** written to `_bmad-output/implementation-artifacts/perf/sprint-1-baseline.md` with:
   - Test environment (Workers preview URL, simulated network profile).
   - 10 individual run timings.
   - p50, p95, p99.
   - Pass/fail vs. 2.5s p95.
   - Screenshot of DevTools Network panel showing parallel requests.

## Acceptance Tests

- [ ] 10 fresh-browser-context Playwright runs against deployed preview; record timings.
- [ ] p95 calculation: sort timings, pick the 95th percentile (rounded up).
- [ ] Manual eyeball of network waterfall to confirm parallelism.

## Dependencies

- **Upstream:** 1.4 (player must work end-to-end).
- **Downstream:** none in Sprint 1. Sprint 4 Story 4.4 extends to 500-concurrent load.

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| (no §16 risk this story; it's the first measurement against G1) | — | This story IS the measurement that determines whether G1 is currently met. |

## Implementation Notes

- **Resource hints in `app/layout.tsx`** (or root `head`):
  ```html
  <link rel="dns-prefetch" href="//cdn.interdependent.tv" />
  <link rel="preconnect" href="https://cdn.interdependent.tv" crossorigin />
  ```
- **Parallel fetch strategy:** Server Component for `app/(tv)/page.tsx` can `await Promise.all([fetch('/api/time'), fetch('/api/channels/channel-1/manifest')])` server-side and pass results to the client provider/player as serializable props (Cat 3 — RSC initial state). This eliminates the client-side fetch waterfall entirely on cold load.
- **hls.js bundle:** dynamic import via `await import('hls.js')` happens in parallel with the manifest fetch on Safari path (which doesn't need hls.js); on Chrome, the bundle is on the critical path — verify it's not a regression vs. a sync import.
- **If p95 > 2.5s on first measurement:** profile DevTools, identify the dominant cost (DNS? KV miss? hls.js parse?), and decide if it's a story (escalate) vs. tuning (within this story's scope).

## Definition of Done

- [ ] AC 1–6 met.
- [ ] Baseline report committed.
- [ ] If p95 fails: a follow-up tuning story is filed BEFORE Sprint 1 is declared exit-ready.
- [ ] Sprint-status.yaml `1-5-cold-load-performance-baseline` → done.
