---
story_id: 4.1
story_key: 4-1-error-states-and-retry
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Error states + retry policy (all pivot §11.5 states)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl) + Sally (UX)"]
effort: M
dependencies: ["1.4", "2.4"]
blocks: []
risks: []
project_context_refs:
  - "Cat 1 (player error retry policy)"
  - "Cat 7 #4 (graceful degradation)"
pivot_doc_refs: ["§11.5 (empty / error states)", "Sprint 4 Story 4.1"]
---

# Story 4.1: Error States + Retry

## User Story

As **a viewer hitting any failure mode**,
I want clear, on-brand error states with sensible auto-retry,
So that transient failures don't break my experience — and persistent failures give me a clear next action.

## Context

Pivot §11.5 enumerates all error states. Story 4.1 implements them with:
- Exponential backoff retry on transient errors (network, segment 404).
- Brand-aligned visuals (velvet-rope aesthetic).
- Clear CTA for unrecoverable states.

## Acceptance Criteria

Implement all 5 states from pivot §11.5:

1. **`/api/channels` fails on cold load:** Full-screen brand frame with "Stand by — we'll be back" message. Auto-retry every 10s. No spinner past 3 attempts; show "Refresh page" CTA.
2. **Channel manifest 404 (mid-session, e.g., channel unpublished):** "This channel is between programs." Auto-redirect to default channel (`channel-1`) after 5s. Counter visible ("Returning to *Channel 1* in 5s...").
3. **HLS segment 404 / network error during playback:**
   - Inline "Reconnecting…" overlay over the video.
   - Auto-retry: 3 attempts with exponential backoff (1s / 2s / 4s — Cat 1 player retry rule).
   - On 3rd failure: render `<PlayerError>` with "Try another channel" CTA → opens guide.
4. **LocalStorage disabled (try/catch on `localStorage.setItem`):** Resume modal silently never renders; session features silently no-op. Viewing still works (Cat 7 #4 graceful degradation).
5. **Mobile data warning (>HD):** DEFERRED per pivot §11.5. Out of Sprint 4 scope. Document the gap in `_bmad-output/implementation-artifacts/known-gaps.md`.

### Component additions

6. `<PlayerError>` component at `app/(tv)/_components/ChannelPlayer/PlayerError.tsx`. Props: `{ reason: string, ctaLabel: string, ctaAction: () => void }`.
7. `<BrandFrame>` component (for state 1) at `app/(tv)/_components/BrandFrame.tsx`. Used for full-screen branded loading/error.
8. `<InlineOverlay>` component for state 3 ("Reconnecting…" over video). Semi-transparent.
9. Toast variants extended (Story 3.3 added basic toast) to support state 2's countdown.

### Retry policy

10. Retry helper at `lib/retry.ts`:
    ```ts
    async function retryWithBackoff<T>(fn: () => Promise<T>, attempts = 3, baseMs = 1000): Promise<T>
    ```
    Backoff: `baseMs * 2^attempt`. Throws on final failure.
11. Used by: `/api/channels` fetch on cold load, manifest fetch, hls.js error handler.

## Acceptance Tests

**Playwright (one per state):**
- [ ] Mock `/api/channels` 500: brand frame renders with "Stand by — we'll be back"; auto-retry every 10s observed.
- [ ] Mock manifest 404 mid-session: "between programs" message + 5s countdown + redirect to `/channel-1`.
- [ ] Mock HLS segment 404: "Reconnecting…" overlay; 3 retry attempts with correct timing; "Try another channel" CTA on final failure.
- [ ] Disable LocalStorage via Playwright context options: Resume Modal never renders; player still works.

## Dependencies

- **Upstream:** 1.4 (player), 2.4 (slug routing).
- **Downstream:** none.

## Risk Alignment

(No direct §16 risks; this story IS the risk mitigation for many failure modes.)

## Implementation Notes

- **`<BrandFrame>` design:** The "velvet-rope" aesthetic — dark background, off-white text, restrained typography. Lives in `_components/` and is reused by Sprint 4 Story 4.2 skeletons.
- **State 2 countdown:** Use `setInterval` with cleanup on unmount. NOT `useServerTime` — this is UI countdown, not schedule math. Real-time is fine here.
- **hls.js retry hook:** `hls.on(Hls.Events.ERROR, (event, data) => { if (data.fatal) { ... retry logic ... } })`. Non-fatal errors hls.js recovers from internally.
- **State 5 documentation:** Create `_bmad-output/implementation-artifacts/known-gaps.md` listing deferred-but-aware items: mobile data warning, accessibility (CC), DRM. This is a parking lot, not a punch list.

## Definition of Done

- [ ] AC 1–11 met.
- [ ] All 4 Playwright state tests green.
- [ ] `known-gaps.md` exists with documented deferrals.
- [ ] Sprint-status.yaml `4-1-error-states-and-retry` → done.
