---
story_id: 3.4
story_key: 3-4-analytics-events
epic: 3
sprint: Sprint 3 — Resume / live decision UX
title: POST /api/events (creator-economy analytics, 204-always contract)
status: ready-for-dev
priority: Should
owners: ["Amelia (impl)"]
effort: M
dependencies: ["3.1", "3.2"]
blocks: []
risks: []
project_context_refs:
  - "Cat 1 (edge runtime, ctx.waitUntil)"
  - "Cat 2 (Zod validation)"
  - "Cat 7 #6 (POST /api/events 204 ALWAYS)"
  - "§17.7 resolution (countryCode payload)"
pivot_doc_refs: ["§9.5 (events endpoint)", "ADR-004 (fire-and-forget analytics)"]
---

# Story 3.4: Analytics Events Endpoint

## User Story

As **the platform's data layer**,
I want a fire-and-forget analytics events endpoint that captures playback signals for creator-economy metrics (NOT ad-targeting),
So that we have empirical viewer-behavior data to inform post-POC creator monetization design.

## Context

Pivot §9.5 contract: `POST /api/events` returns **204 unconditionally**. Postgres write failures are caught + logged + swallowed (Cat 7 #6). An agent writing a naive `await db.insert(...)` without try/catch will surface 5xx to the client, which BLOCKS PLAYBACK because the client fires `start` events at playback init. This is the single most likely bug to crash the demo.

Victor's §17 flag: design analytics for the actual monetization model (creator economy), not generic ad-targeting taxonomy. The event types reflect viewer engagement, not ad impression hooks.

§17.7 resolution: include `countryCode` (from `CF-IPCountry` Workers header) in event payload — geo-aware analytics, NOT geo-fencing.

**Two-layer analytics architecture (Plan C consensus, 2026-05-12 party):** This pipeline is the **authoritative source for creator-economy metrics** — completion rate per asset, return-visitor rate per creator, channel loyalty, cohort retention, time-of-day engagement, geo distribution. Story 1.4 separately wires the **Mux Data SDK** for **delivery QoE telemetry** (startup time, rebuffer ratio, playback failures). The two pipelines are complementary, not competing: Mux Data tells us whether the *player* is healthy; `playback_events` tells us whether *creators are building audiences*. **Vendor selection (Mux Live vs Cloudflare Stream Live) for the post-POC live creator channel is informed by Story 4.7's docs-and-free-trial evaluation, NOT by these dashboards** — POC sample size is too small to drive a defensible production-platform decision from this data.

## Acceptance Criteria

### Endpoint

1. Path: `app/api/events/route.ts`. Exports `POST` handler.
2. `export const runtime = 'edge'`.
3. **Response is `new Response(null, { status: 204 })` UNCONDITIONALLY** — including on Zod parse failure, Postgres failure, or any internal exception. Per Cat 7 #6.
4. Request body validated with Zod:
   ```ts
   const EventsBatch = z.object({
     sessionId: z.string().uuid(),
     events: z.array(z.object({
       type: z.enum(['start', 'pause', 'resume', 'heartbeat', 'channel_change', 'vod_detour_enter', 'vod_detour_exit', 'end']),
       channelId: z.string().uuid().nullable(),
       assetId: z.string().uuid().nullable(),
       positionSec: z.number().int().nullable(),
       tsMs: z.number().int(),
     })).max(20),
   });
   ```
5. On valid body: insert events into `playback_events` table. Each row gets `countryCode` populated from `request.cf.country` (Cloudflare Workers header).
6. **DB write uses `ctx.waitUntil()`** (Cat 1 + 2) so the 204 returns BEFORE the Postgres insert completes. Fire-and-forget per ADR-004.
7. **Try/catch around the DB write** inside the `waitUntil` callback. On failure: `console.error(...)` only. NEVER throw to the route handler.

### Event types (Victor's creator-economy framing)

8. Event types emitted by client:
   - `start` (fired when player first plays a channel session)
   - `pause` (fired on user-initiated pause)
   - `resume` (fired on play after pause)
   - `heartbeat` (every 10s during active playback — captures "real watch time")
   - `channel_change` (slug-to-slug switch)
   - `vod_detour_enter` (from Resume Modal Continue)
   - `vod_detour_exit` (asset ended OR early exit)
   - `end` (player unmount; `visibilitychange → hidden`)
9. Each event carries `tsMs` (client-side timestamp from `getServerNow()` — Cat 7 #2 — not raw `Date.now()`).
10. `heartbeat` includes `positionSec` so completion-rate analytics can compute "fraction of asset viewed" downstream.

### Client batching

11. Client batches events: up to 20 events OR 5 seconds, whichever first.
12. Batch send via `fetch('/api/events', { method: 'POST', body: JSON.stringify(batch), keepalive: true })`. `keepalive: true` is critical for `beforeunload` and `visibilitychange → hidden` flushes.
13. Failed batches are dropped (best-effort). NOT retried. NOT queued in LocalStorage.

### Geo-aware analytics (§17.7 resolution)

14. `countryCode` is populated server-side from `request.cf?.country` (string, 2-letter ISO). If missing (local dev), store `null`.
15. **This is NOT geo-fencing.** No request is denied based on country. Captured for downstream analysis only.

## Acceptance Tests

**Edge-runtime tests:**
- [ ] POST `/api/events` with valid body returns 204.
- [ ] POST with malformed body returns 204 (still — per AC 3). Verify Zod failure logged via test capture.
- [ ] POST with valid body inserts N rows into `playback_events` (verify via Drizzle query post-flush).
- [ ] Mock Postgres to throw: route still returns 204; error logged to `console.error`.
- [ ] `countryCode` is populated from a mocked `request.cf.country = 'US'` test fixture.

**Client batching unit tests:**
- [ ] Calling `track('start')` 5× within 1s queues 5 events; nothing fires yet.
- [ ] After 5s timer expires, single batch POSTs with 5 events.
- [ ] After 20 events queue, batch fires immediately regardless of timer.
- [ ] `beforeunload` triggers immediate flush of pending batch.

## Dependencies

- **Upstream:** 3.1, 3.2 (modes that emit events). Also 1.1 (`playback_events` table with `countryCode`).
- **Downstream:** Sprint 4 / scale-sprint analytics pipeline (out of scope; reads from Postgres or migrates to Cloudflare Analytics Engine).

## Risk Alignment

| ID | Risk | Mitigation in this story |
|---|---|---|
| (No §16 risk directly; this story implements the 204-always contract that PREVENTS analytics failures from crashing playback.) | — | Cat 7 #6 is the rule; AC 3 + 7 are the enforcement. |

## Implementation Notes

- **`ctx.waitUntil(promise)` pattern** in Workers/OpenNext:
  ```ts
  export async function POST(request: Request) {
    const ctx = getRequestContext(); // OpenNext platform context
    const parsed = EventsBatch.safeParse(await request.json().catch(() => null));
    if (!parsed.success) return new Response(null, { status: 204 });
    ctx.waitUntil(insertEvents(parsed.data, request.cf?.country).catch(console.error));
    return new Response(null, { status: 204 });
  }
  ```
  Order matters: parse → schedule the work → return. The handler returns immediately; the DB insert happens after the response is sent.
- **Drizzle batched insert:** use `db.insert(playbackEvents).values([...rows])` for a single multi-row INSERT. Faster than N inserts.
- **Client-side batching helper** at `lib/analytics.ts`:
  ```ts
  let queue: Event[] = [];
  let timer: number | null = null;
  export function track(event: Event) {
    queue.push(event);
    if (queue.length >= 20) flush();
    else if (!timer) timer = window.setTimeout(flush, 5000);
  }
  function flush() { /* fetch with keepalive; clear queue */ }
  window.addEventListener('beforeunload', flush);
  document.addEventListener('visibilitychange', () => { if (document.hidden) flush(); });
  ```
- **`request.cf` typing:** Cloudflare Workers extends `Request` with a `.cf` object. Type-narrow safely:
  ```ts
  const country = (request as Request & { cf?: { country?: string } }).cf?.country ?? null;
  ```

## Definition of Done

- [ ] AC 1–15 met.
- [ ] 204-always contract verified by failure-injection test.
- [ ] Client batching tests pass.
- [ ] `countryCode` populated in test inserts.
- [ ] **Two-pipeline architecture documented:** this story = creator-economy events; Story 1.4 = Mux Data QoE telemetry. README cross-link in `docs/ANALYTICS.md` (single page, ~half-page).
- [ ] Sprint-status.yaml `3-4-analytics-events` → done.
