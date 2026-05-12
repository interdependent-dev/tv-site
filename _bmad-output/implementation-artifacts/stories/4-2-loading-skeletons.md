---
story_id: 4.2
story_key: 4-2-loading-skeletons
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Loading skeletons in velvet-rope aesthetic (no FOUC)
status: ready-for-dev
priority: Must
owners: ["Sally (UX) + Amelia (impl)"]
effort: M
dependencies: ["2.4", "4.1"]
blocks: []
risks: []
project_context_refs:
  - "Cat 5 (Tailwind, no CSS-in-JS, clsx for conditionals)"
pivot_doc_refs: ["§11.4 (player chrome)", "Sprint 4 Story 4.2"]
---

# Story 4.2: Loading Skeletons (Velvet-Rope Aesthetic)

## User Story

As **a viewer experiencing any loading state**,
I want brand-aligned skeleton placeholders instead of flashes of unstyled content,
So that the platform feels polished and intentional during the unavoidable network gaps.

## Context

"Velvet rope" is the platform-wide UX metaphor (pivot §19 glossary). Loading skeletons should feel like the curtain before the show — restrained, dark, with subtle animated detail. NOT generic gray-box skeletons.

## Acceptance Criteria

1. **`<ChannelGuide>` skeleton:** while initial `/api/channels` fetch is pending, render 5 placeholder tile shapes with subtle pulse animation. Same layout dimensions as actual tiles (prevents layout shift).
2. **`<ChannelPlayer>` skeleton:** while manifest fetch + hls.js parse are pending, render a dark frame with the channel logo (if known) centered. No spinner; subtle pulse only.
3. **`<ResumeModal>` skeleton:** N/A (modal doesn't render until session + manifest are loaded; no skeleton needed).
4. **No FOUC anywhere.** First paint either shows a skeleton or final content, never half-styled HTML.
5. **Skeletons use Tailwind utility classes** (per Cat 5). No new CSS files except the global token sheet.
6. **Subtle pulse animation:** `animate-pulse` Tailwind utility on skeleton elements. Disable in `prefers-reduced-motion: reduce` media query.
7. **Brand tokens used:** dark background (defined in `tailwind.config.ts`), restrained typography, no accent colors in skeleton state.
8. **Performance:** skeletons must not add measurable cold-load time vs. Sprint 1 baseline (Story 1.5). Re-run baseline; assert p95 still ≤ 2.5s.

## Acceptance Tests

- [ ] Throttle `/api/channels` to 3s in Playwright: skeleton renders for full 3s; no FOUC.
- [ ] Visual regression: Playwright snapshot of skeleton state for Guide + Player. Compared with target design (or visual review by Sally).
- [ ] `prefers-reduced-motion: reduce` set: assert no pulse animation.

## Dependencies

- **Upstream:** 2.4 (player + guide both consume skeletons), 4.1 (BrandFrame component is the precedent).
- **Downstream:** none.

## Risk Alignment

(No §16 risk; UX polish.)

## Implementation Notes

- **Skeleton component pattern:**
  ```tsx
  function ChannelTileSkeleton() {
    return (
      <div className="h-32 w-60 rounded-lg bg-zinc-900 animate-pulse motion-reduce:animate-none" />
    );
  }
  ```
- **Layout-shift prevention:** skeleton dimensions must match real tile/player dimensions exactly. Use Tailwind `aspect-video` or fixed pixel dimensions to lock the box.
- **`<Suspense>` boundaries:** Next.js App Router uses `loading.tsx` for route-level loading. Place skeleton-only content in `app/(tv)/loading.tsx` and `app/(tv)/[slug]/loading.tsx`.
- **Re-run Story 1.5's perf measurement** to ensure skeleton render doesn't push past G1 budget.

## Definition of Done

- [ ] AC 1–8 met.
- [ ] Visual regression passes.
- [ ] Reduced-motion respected.
- [ ] Perf baseline re-verified.
- [ ] Sprint-status.yaml `4-2-loading-skeletons` → done.
