---
story_id: 2.2
story_key: 2-2-get-api-channels-list-endpoint
epic: 2
sprint: Sprint 2 — Multi-channel and channel guide
title: GET /api/channels (list endpoint with current/next per channel)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: M
dependencies: ["2.1"]
blocks: ["2.3"]
risks: []
project_context_refs:
  - "Cat 1 (edge runtime)"
  - "Cat 3 (KV TTL 30s + SWR; Drizzle O(channels) query)"
  - "Cat 7 #9 (runtime = edge)"
pivot_doc_refs: ["§9.2 (channels list)", "§12.1 (KV TTLs)"]
---

# Story 2.2: GET /api/channels

## User Story

As **the `<ChannelGuide>` (Story 2.3)**,
I want a single endpoint that returns the full list of published channels with current + next program metadata,
So that the guide can render the multi-channel row without N+1 query waterfalls.

## Context

The `current` and `next` program per channel is computed via the same algorithm as `currentProgram` (pivot §7.2) but at the API layer for the list shape. Single query, O(channels) — not per-asset N+1.

KV TTL = 30s with `stale-while-revalidate=300` (pivot §12.1). Tag-purge on any channel publish/unpublish.

## Acceptance Criteria

1. Path: `app/api/channels/route.ts`. Exports `GET` handler.
2. `export const runtime = 'edge'`.
3. Response 200 shape (pivot §9.2):
   ```json
   {
     "serverTsMs": <number>,
     "channels": [
       {
         "id": "<uuid>",
         "slug": "channel-1",
         "name": "...",
         "logoUrl": "...",
         "current": { "assetId": "<uuid>", "title": "...", "startsAtMs": <ms>, "endsAtMs": <ms> },
         "next":    { "assetId": "<uuid>", "title": "...", "startsAtMs": <ms>, "endsAtMs": <ms> }
       }
     ]
   }
   ```
4. **`current` and `next` are computed via the SAME algorithm as `currentProgram`** (pivot §7.2 + Cat 7 #2). Reuse `currentProgram(nowMs, channel)` — do not re-implement.
5. **Single query / O(channels):** Drizzle query returns all channels + their `channel_assets` + linked `assets` rows in one round-trip via a join. NO loop over channels with per-channel queries.
6. **`startsAtMs` and `endsAtMs`** are computed from `epoch_anchor + cumulative_offset_in_loop_so_far` for the current iteration. Both serialized as `*Ms` numbers (Cat 2).
7. KV-cached via `cached('channels:list', 'channels:list', 30, loader)`.
8. Headers: `Cache-Control: public, max-age=30, stale-while-revalidate=300`.
9. Query function at `db/queries/channels.ts` as `getPublishedChannelsWithSchedule()`. Returns plain serializable rows.
10. Route handler imports the query, computes `current`/`next` per channel, returns shape. Never `db` directly.

## Acceptance Tests

**Edge-runtime tests:**
- [ ] GET `/api/channels` returns 200 with 5 channels (sorted by `sort_order`).
- [ ] Each channel has a `current` and `next` block.
- [ ] `current.startsAtMs <= serverTsMs < current.endsAtMs`.
- [ ] `next.startsAtMs === current.endsAtMs` (back-to-back continuity).

**Drizzle query test:**
- [ ] `getPublishedChannelsWithSchedule()` is O(1) round-trips. Verify by EXPLAIN ANALYZE or query log inspection — no `N+1` pattern.

**Performance (carry into 2.5):**
- [ ] Cold KV miss → DB → response: p95 < 200ms against Neon branch from Workers preview.
- [ ] KV hit: p95 < 30ms.

## Dependencies

- **Upstream:** 2.1 (≥2 published channels).
- **Downstream:** 2.3 (Guide consumes), 2.5 (concurrency test exercises).

## Risk Alignment

(No direct §16 risk; performance concerns roll into 2.5 + 4.4.)

## Implementation Notes

- **`current`/`next` computation:** for each channel, compute `positionInLoop = ((nowMs - epochAnchorMs) / 1000) % loopDurationSec` (with double-mod). Then walk the playlist cumulatively to find the current asset and the next one. Pure function.
- **Reuse the schedule helper:** if `currentProgram` lives at `lib/schedule.ts` (extracted from Story 1.4), import it. Otherwise extract during this story — cross-component reuse warrants it.
- **`endsAtMs` for `current`:** `epochAnchorMs + (cumulative_offset_to_end_of_current_asset_in_this_iteration_of_the_loop)`. The iteration count is `Math.floor(elapsedSec / loopDurationSec)`.
- **`next` across loop boundary:** if current is the last asset in the playlist, `next` is `playlist[0]` of the next iteration (different start time).
- **Tag-purge plan:** any future channel-publish/unpublish admin flow MUST call `purgeTag('channels:list')` after the DB write.

## Definition of Done

- [ ] AC 1–10 met.
- [ ] Tests green (edge pool + Neon branch).
- [ ] O(channels) query verified.
- [ ] Sprint-status.yaml `2-2-get-api-channels-list-endpoint` → done.
