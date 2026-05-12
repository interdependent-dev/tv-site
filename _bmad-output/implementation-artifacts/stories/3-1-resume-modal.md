---
story_id: 3.1
story_key: 3-1-resume-modal
epic: 3
sprint: Sprint 3 — Resume / live decision UX
title: Resume modal with 5-row decision matrix (pivot §11.2)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl) + Sally (UX review)"]
effort: M
dependencies: ["2.4", "1.3"]
blocks: ["3.2", "3.3"]
risks: []
project_context_refs:
  - "Cat 3 (route structure — modal on / ONLY)"
  - "Cat 7 #10 (resume modal decision matrix is the spec)"
pivot_doc_refs: ["§11.2 (decision matrix)", "§10.6 (resume modal)", "§6.5 (return-visit sequence)"]
---

# Story 3.1: Resume Modal

## User Story

As **a returning viewer with a recent session (<24h)**,
I want a modal that offers me either to continue what I was watching (VOD-style) or jump to the live channel,
So that "coming back" works in the intuitive sense — and the decision is explicit, not magical.

## Context

Pivot §11.2 has a 5-row decision matrix mapping session state to (copy, button-1, button-2). **The matrix is the spec.** One Playwright test per row. Don't invent new states.

**Mount on `/` ONLY** (project-context.md Cat 7 #10; pivot §10.6). Mounting in `layout.tsx` violates the contract because it would show on `/[slug]` too.

Session age >24h: skip modal entirely; autoplay live on `lastChannel`. NOT configurable.

## Acceptance Criteria

### Component
1. Component at `app/(tv)/_components/ResumeModal/index.tsx`. Client component.
2. **Mounts on `/` ONLY.** `app/(tv)/page.tsx` includes `<ResumeModal />`; `[slug]/page.tsx` does NOT.

### Decision matrix (pivot §11.2 — implement EXACTLY)

3. **Row 1: Same asset still live on channel; age < 30 min:**
   - Copy: "Welcome back. Pick up where you left off?"
   - Button 1: "Continue *Title* — Y:ZZ" (where Y:ZZ = formatted `lastPositionSec` mm:ss)
   - Button 2: "Jump to live"
4. **Row 2: Same asset still live; age 30 min – 24h:**
   - Copy: "You were watching *Title*."
   - Button 1: "Continue (VOD) — Y:ZZ"
   - Button 2: "Go live on *Channel*"
5. **Row 3: Different asset live now; age <24h:**
   - Copy: "You were watching *Title*."
   - Button 1: "Continue *Title* (VOD)"
   - Button 2: "Go live on *Channel* (now *NowTitle*)"
6. **Row 4: Channel no longer published:**
   - Copy: "That channel has moved on."
   - Button 1: "Continue *Title* (VOD)"
   - Button 2: "Browse channels"
7. **Row 5: Asset no longer available:**
   - Copy: "Welcome back."
   - Single CTA: "Browse channels"

### Branching logic

8. Read `session` from `useSession` on mount. If `null`, do NOT render.
9. Compute `ageMs = serverNow - session.exitedAtMs`. If `> 24h`, do NOT render; instead fire an effect that auto-plays live on `session.lastChannelSlug` (cold-entry-with-stale-session behavior).
10. If session valid and <24h: fetch `/api/channels/{session.lastChannelSlug}/manifest` (likely cached). Compute the currently-live asset using `currentProgram(serverNow, manifest)`. Compare with `session.lastAssetId`. Decide which row applies.
11. **Channel-unpublished branch (row 4):** 404 on the manifest fetch → channel is gone. Per **D2-VOD-URL resolution (2026-05-12 party consensus)**, the session itself is canonical for the VOD detour path: read `session.lastAssetHlsUrl` and `session.lastAssetTitle` directly (cached at exit time by Story 1.4 / 2.4). No manifest-by-id endpoint exists in POC. If those session fields are missing (session predates the field — pre-Sprint-3 deploys, or schema-mismatch reset), degrade to row 5.
12. **Asset-unavailable branch (row 5):** session lacks `lastAssetHlsUrl` (the canonical VOD source) — typically because the session was cleared, was written before the field landed, or LocalStorage was tampered with. Degrade to single CTA.

### Behavior

13. **Button 1 (Continue):** depending on row, either:
    - Row 1: `router.push('/{lastChannelSlug}')` with a query param `?resumeMode=live-position`. Player sees this, seeks to `lastPositionSec` once. (Same-asset-still-live → just resume the live channel; the live position likely aligns within 30 min.)
    - Rows 2/3/4: navigate to a VOD detour route — Story 3.2's responsibility. For Story 3.1, dispatch a state event (`PlayerMode → { kind: 'vod-detour', assetId, resumeMs, returnToSlug }`) that Story 3.2 consumes.
14. **Button 2 (Live):** `router.push('/{lastChannelSlug}')`. No resume; live position. Modal dismisses.
15. **Modal is dismissable** (Esc key, click outside, button activation). Never re-renders in the same session.
16. **Keyboard accessible:** Tab cycles buttons; Enter activates focused button; Esc closes (treating Esc as Button 2 / live — the more "normal" choice).
17. **Mobile:** large tap targets (≥44pt). Full-width buttons stacked.
18. **Desktop:** side-by-side buttons.

### State / persistence

19. Once shown and dismissed (either CTA), set a session-scoped flag (`useSession.updateSession({ resumeModalShown: true })`) so refreshing `/` doesn't re-show it within the same browser session.

## Acceptance Tests

**Playwright (one test per row of the matrix — pivot §11.2 EXACTLY):**

- [ ] Row 1 test: session with same-asset-live, age 5 min → modal renders Row 1 copy; Continue button shows formatted time; Live button works.
- [ ] Row 2 test: session with same-asset-live, age 2h → modal renders Row 2 copy.
- [ ] Row 3 test: session with different-asset-live-now, age 5h → modal renders Row 3 copy with both asset titles correct.
- [ ] Row 4 test: session with unpublished channel → modal renders Row 4 copy + "Browse channels" CTA.
- [ ] Row 5 test: session with deleted asset → modal renders Row 5 single CTA.

**Edge cases:**
- [ ] Age > 24h: modal does NOT render; auto-plays live on `lastChannelSlug`.
- [ ] No session: modal does NOT render.
- [ ] Modal on `/channel-1`: does NOT render (mount-on-`/`-only rule).

## Dependencies

- **Upstream:** 2.4 (slug routing), 1.3 (`useSession`).
- **Downstream:** 3.2 (VOD detour player mode handles the Continue button for rows 2–4), 3.3 (edge cases handle rows 4–5 fallbacks).

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| R6 | LocalStorage cleared by viewer | Modal does NOT render with null session → cold entry. Pivot R6 = Closed. |

## Implementation Notes

- **Row decision is a pure function** of `(session, serverNow, manifest)`. Extract to `lib/resume-decision.ts`. Each row is a discriminated union variant:
  ```ts
  type ResumeRow =
    | { row: 1; copy: ..., btn1: ..., btn2: ... }
    | ...
    | { row: 5; copy: ..., btn1: ... };
  ```
- **`formatTime(positionSec)` helper:** mm:ss format, e.g., 1234 → "20:34". No library; one-line function.
- **Manifest 404 (row 4 detection):** wrap the manifest fetch in try/catch; on 404, return row 4 directly.
- **Title + URL lookup for VOD detour (rows 2/3/4):** per **D2-VOD-URL consensus**, session is canonical. Extend `SessionV1` schema (additively, per D3 consensus) in this story with `lastAssetTitle: string | null` and `lastAssetHlsUrl: string | null`. Story 1.4 / 2.4's `updateSession` call sites add these to every session write. Resume Modal reads them directly — never refetches a manifest to compute a title.
- **Schema additions in this story (D3 = additive evolution):**
  - `lastAssetTitle: string | null` — cached at exit; nullable for legacy sessions.
  - `lastAssetHlsUrl: string | null` — cached at exit; the VOD detour source URL.
  - `resumeModalShown: boolean` (AC 19) — session-scoped flag, defaults `false`.
- **Modal a11y:** use a `role="dialog"` + `aria-modal="true"` + `aria-labelledby` pointing to the heading. Focus trap within the modal while open. Return focus to trigger on close.

## Definition of Done

- [ ] AC 1–19 met.
- [ ] All 5 Playwright row tests green.
- [ ] Edge-case tests green.
- [ ] Sprint-status.yaml `3-1-resume-modal` → done.
