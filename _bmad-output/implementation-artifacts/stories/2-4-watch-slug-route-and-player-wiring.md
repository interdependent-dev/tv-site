---
story_id: 2.4
story_key: 2-4-watch-slug-route-and-player-wiring
epic: 2
sprint: Sprint 2 — Multi-channel and channel guide
title: /[slug] apex route + player swap on channel change
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: M
dependencies: ["2.3", "1.4"]
blocks: ["3.1"]
risks: []
project_context_refs:
  - "Cat 3 (Route structure, URL = state contract, slug validation)"
  - "Cat 7 #12 (hls.js teardown), #13 (two code paths), #14 (eventually consistent)"
pivot_doc_refs: ["§10.1 (component tree)", "§3 G2 (switch latency ≤500ms)"]
---

# Story 2.4: /[slug] Apex Route + Player Wiring

## User Story

As **a viewer who switches channels via the guide OR pastes a channel URL**,
I want clean apex URLs (`interdependent.tv/channel-1`) that load directly into the player without page reload,
So that switching is fast (G2: ≤500ms p95) and channel URLs are shareable.

## Context

D14 settled the URL shape: viewer URLs live on the apex (`interdependent.tv/channel-1`), NOT behind a `/watch/` prefix. This requires:
- Next.js dynamic route at `app/(tv)/[slug]/page.tsx`.
- Slug validation at the route handler to prevent the apex `[slug]` route from catching unrelated paths like `/about`, `/manifest.json`, future top-level routes.

The route handler validates the slug against the channels table; non-channel paths return `notFound()`.

This story also implements the hls.js teardown discipline on channel change (Cat 7 #12) — the memory-leak test is THIS story's AC, not Story 1.4's (though 1.4 implements the teardown helper).

## Acceptance Criteria

1. **Route file at `app/(tv)/[slug]/page.tsx`.** Server Component.
2. **Slug validation:** the page reads the slug param, queries `db/queries/channels.ts::getChannelBySlug(slug)`. If null or `is_published = 0`, calls Next.js `notFound()`.
3. **`not-found.tsx`** at `app/(tv)/not-found.tsx` renders a "Channel not found" message with a "Browse channels" CTA back to `/`.
4. **The page renders `<ChannelPlayer slug={slug} initialManifest={manifest} />`** + `<ChannelGuide />`. `initialManifest` is the server-fetched manifest passed as a serializable prop (Cat 3 RSC pattern; eliminates client fetch waterfall).
5. **Channel switch via `<ChannelGuide>` click** uses `router.push('/${newSlug}')`. NOT `window.location.assign`. NOT full reload.
6. **Player tears down hls.js cleanly on slug change.** Verified by AC 7 memory-leak test.
7. **Memory-leak test (Cat 4 — Playwright + DevTools CDP):**
   - Switch channels 10× in sequence (via `<ChannelGuide>` clicks).
   - After each switch, query DevTools heap: assert exactly 1 active `<video>` element.
   - After the 10 switches, assert 0 detached `HTMLMediaElement` instances in the heap.
8. **LocalStorage updates on channel change:** `useSession.updateSession({ lastChannelId, lastChannelSlug, lastAssetId, lastAssetHlsUrl, lastAssetTitle, lastPositionSec, exitedAtMs })` fires within 100ms of `router.push`. Per **D2-VOD-URL consensus**, `lastAssetHlsUrl` + `lastAssetTitle` are pulled from the outgoing channel's current-asset entry — required for downstream Resume Modal + VOD detour (Sprint 3).
9. **Shareable URL test:** Opening `https://interdependent.tv/channel-3` from a fresh browser context renders identically to clicking `channel-3` from `/channel-1`. No detectable behavior difference.
10. **Switch latency:** from `router.push` call to player visibly playing the new channel ≤ 500ms p95 on broadband (Cat 7 G2 + verified in 2.5).

## Acceptance Tests

**Playwright (THE Sprint 2 demo gate):**
- [ ] Open two browser windows on different channels → both play simultaneously (Sprint 2 exit criterion).
- [ ] Open `/channel-3` from fresh context → renders identically to switching from `/channel-1` via guide.
- [ ] Memory-leak test green (10 switches, 1 active video, 0 detached HTMLMediaElement).
- [ ] Switch latency (p95 of 10 runs) ≤ 500ms.
- [ ] Navigate to `/not-a-real-channel` → renders `not-found.tsx`.

## Dependencies

- **Upstream:** 2.3 (Guide drives channel switches), 1.4 (Player).
- **Downstream:** 3.1 (Resume modal mounts on `/`, but its decisions involve channel slugs from this story).

## Risk Alignment

| ID | Risk | Mitigation in this story |
|---|---|---|
| R8 | Schedule edit during heavy viewing | Manifest re-fetch (60s background, in 1.4) + clean tear-down enables graceful schedule-edit mid-watch. |

## Implementation Notes

- **Slug validation pattern (server-side):**
  ```ts
  // app/(tv)/[slug]/page.tsx
  export default async function ChannelPage({ params }: { params: { slug: string } }) {
    const channel = await getChannelBySlug(params.slug);
    if (!channel) notFound();
    const manifest = await getChannelManifestBySlug(params.slug);
    return (
      <>
        <ChannelPlayer slug={params.slug} initialManifest={manifest} />
        <ChannelGuide />
      </>
    );
  }
  ```
  This avoids a separate API hop on initial render — the Server Component reads the DB once and serves the manifest as a prop to the client `<ChannelPlayer>` (Story 1.4) and `<ChannelGuide>` (Story 2.3).
- **`<ChannelGuide>` link**: use `next/link` (`<Link href={`/${slug}`}>`) for prefetch, OR `router.push` in click handler — pick one consistently. `<Link>` enables Next.js's automatic prefetch on hover, which helps p95 latency.
- **Memory-leak test setup:** Playwright's CDP support lets you query `Performance.getMetrics()` for "Nodes" and "JSHeapUsedSize". Detached element detection requires a more involved heap snapshot — feasible with `client.send('HeapProfiler.takeHeapSnapshot')` + post-processing. If overkill for POC, an acceptable proxy: assert `document.querySelectorAll('video').length === 1` after each switch.

## Definition of Done

- [ ] AC 1–10 met.
- [ ] Memory-leak test green (or proxied with `querySelectorAll` count + dev review of cleanup logic).
- [ ] Switch latency p95 ≤ 500ms documented.
- [ ] Sprint-status.yaml `2-4-watch-slug-route-and-player-wiring` → done.
