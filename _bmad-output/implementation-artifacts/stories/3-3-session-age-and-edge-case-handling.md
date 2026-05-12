---
story_id: 3.3
story_key: 3-3-session-age-and-edge-case-handling
epic: 3
sprint: Sprint 3 — Resume / live decision UX
title: Session age + resume edge case handling
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: S
dependencies: ["3.1"]
blocks: []
risks: []
project_context_refs:
  - "Cat 3 (useSession contract)"
  - "Cat 7 #4 (LocalStorage only viewer state)"
pivot_doc_refs: ["§11.2 (decision matrix rows 4+5)", "§11.1 (entry flow)"]
---

# Story 3.3: Session Age + Resume Edge Case Handling

## User Story

As **a returning viewer with various edge-case session states**,
I want the platform to handle stale sessions, unpublished channels, and deleted assets gracefully,
So that no error states leak to me — I either get my content, or I get a clear fallback.

## Context

This story makes the Resume Modal's edge-case branches actually work. Story 3.1 implemented the matrix; this story implements the underlying handlers (channel-unpublished fallback, asset-deleted fallback, >24h auto-live).

## Acceptance Criteria

### Stale session (>24h)

1. On `/` mount: if `serverNow - session.exitedAtMs > 24 * 60 * 60 * 1000`, **session is ignored**; auto-play live on `session.lastChannelSlug`.
2. **Do NOT** render the Resume Modal.
3. If `session.lastChannelSlug` is also unpublished/missing: fall back to **default channel** (`channel-1`); show a non-blocking toast: "Continuing on *Channel 1*."

### Channel no longer published (Row 4 of §11.2)

4. Resume Modal Row 4 is triggered when `/api/channels/{lastChannelSlug}/manifest` returns 404 OR the channel's `is_published` flag is 0.
5. Continue button (Row 4) plays the asset standalone as VOD. Per **D2-VOD-URL party consensus (2026-05-12)**, the VOD source is **session-canonical**: pass `session.lastAssetHlsUrl` + `session.lastAssetTitle` (cached at exit by Story 1.4 / 2.4) into PlayerMode `vod-detour`. No `/api/assets/{id}` endpoint, no manifest-by-id lookup. If `session.lastAssetHlsUrl` is null (legacy session predating the field), Row 4 degrades to Row 5.
6. Browse channels button navigates to `/` and dismisses modal.

### Asset deleted / canonical URL missing (Row 5)

7. Detection: `session.lastAssetHlsUrl` is null/missing (legacy session, cleared LocalStorage, or session-shape mismatch). This is the only Row 5 detection — there's no separate "is this asset still alive" check in POC, because we trust the cached URL.
8. Modal degrades to single CTA: "Browse channels" → navigates to `/` (and dismisses).

### Session-shape mismatch (corrupt / outdated `itv.session.v1`)

9. `useSession` Zod validation fails on read → `clearSession()` (Story 1.3 already enforces). UI treats as cold entry → autoplay default channel.
10. **No crash UI.** Console.warn for debug but no visible error.

### Non-blocking toast helper

11. Toast component at `app/(tv)/_components/Toast.tsx` (or similar). Single-line, top of viewport, auto-dismiss after 4s. Used by AC 3.
12. Toast variants: `info` (gray), `warn` (yellow), `error` (red). For POC, only `info` and `warn` are needed.

## Acceptance Tests

**Playwright:**
- [ ] Stale session (>24h): no modal; autoplay last channel.
- [ ] Stale session + last channel unpublished: autoplay `channel-1` with toast.
- [ ] Channel unpublished (current behavior): modal Row 4; Continue plays VOD; Browse navigates to `/`.
- [ ] Asset deleted (current behavior): modal Row 5; only Browse CTA.
- [ ] Corrupt LocalStorage: cold entry behavior; no error visible.

**Unit:**
- [ ] `decideResumeRow(session, serverNow, manifest)` pure function covers all 5 + stale + corrupt cases.

## Dependencies

- **Upstream:** 3.1 (Resume Modal).
- **Downstream:** none in Sprint 3.

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| R6 | LocalStorage cleared / corrupt | This story formalizes the graceful-degradation logic. Pivot R6 = Closed; this story is the enforcement. |

## Implementation Notes

- **24h constant:** define as `const SESSION_STALE_MS = 24 * 60 * 60 * 1000` in `lib/session-rules.ts`. Single source of truth.
- **Toast trigger from session-stale-with-unpublished-channel branch:** `<ResumeModal>` is the wrong place (the modal doesn't render in this branch). The home page (`app/(tv)/page.tsx`) reads session in a server component → passes computed initial state to client → client decides modal vs. toast. Toast is fired via a top-level context (`<ToastProvider>` in `layout.tsx`).
- **`is_published` check at manifest endpoint:** Story 1.2 already 404s when `is_published = 0`. Row 4 detection therefore = manifest 404. No extra logic needed; just exposed in the resume-decision function.
- **Session-canonical VOD source (D2-VOD-URL consensus):** The Row 4 → VOD path relies on session-cached `lastAssetHlsUrl` + `lastAssetTitle`. This requires Story 1.4 and 2.4 to write those fields on every session save. Cross-reference added to both stories. If you find yourself fetching a manifest to compute a VOD title or URL, you've drifted from the resolved pattern — fix the call site, not the schema.

## Definition of Done

- [ ] AC 1–12 met.
- [ ] Playwright tests pass.
- [ ] Sprint-status.yaml `3-3-session-age-and-edge-case-handling` → done.
