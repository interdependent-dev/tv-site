---
story_id: 1.2
story_key: 1-2-api-time-and-channel-manifest-endpoints
epic: 1
sprint: Sprint 1 — Schedule and single-channel playback
title: GET /api/time and GET /api/channels/[slug]/manifest
status: ready-for-dev
priority: Must
owners: ["Amelia (impl)"]
effort: M
dependencies: ["1.1"]
blocks: ["1.3", "1.4"]
risks: []
project_context_refs:
  - "Cat 1 (Edge runtime contract)"
  - "Cat 2 (Zod trust-boundary validation)"
  - "Cat 3 (KV access patterns, cached() helper)"
  - "Cat 7 #6 (POST /api/events 204 rule — not this story but cross-aware)"
  - "Cat 7 #9 (export const runtime = 'edge')"
pivot_doc_refs: ["§9.1 (/api/time)", "§9.3 (/api/channels/[slug]/manifest)", "§12.1 (KV TTLs)"]
---

# Story 1.2: GET /api/time and GET /api/channels/[slug]/manifest

## User Story

As **the `<ChannelPlayer>` (Story 1.4 client)**,
I want an authoritative server time endpoint and a per-channel manifest endpoint,
So that the player can compute the deterministic "live" position from server-anchored wall-clock time + the channel's playlist.

## Context

These are the two read endpoints Story 1.4's player consumes on every cold load. Both are KV-cached read-through; both export `runtime = 'edge'`. `/api/time` additionally exports `dynamic = 'force-dynamic'` to prevent build-time `Date.now()` static rendering (project-context.md Cat 1 + Cat 7 #9).

The `cached(key, tag, ttl, loader)` helper (project-context.md Cat 3) is introduced in this story. ALL future read-through caching MUST flow through it. Inline `env.KV.get` is banned.

## Acceptance Criteria

### Common
1. Both routes export `export const runtime = 'edge'` (Cat 7 #9). Build fails if omitted.
2. `cached(key: string, tag: string, ttlSec: number, loader: () => Promise<T>): Promise<T>` helper exists at `lib/cache.ts`. Behavior:
   - Reads `env.KV.get(key)`. If hit, returns parsed JSON.
   - If miss, calls `loader()`, writes via `env.KV.put(key, JSON.stringify(value), { expirationTtl: ttlSec })`, updates tag index, returns value.
   - Tag index: `tag:<tag>` stores a JSON array of keys; on each put, the key is appended (deduplicated).
3. `purgeTag(tag: string)`: reads `tag:<tag>` → deletes all listed keys → deletes the tag index entry. Idempotent.
4. Both routes validate request params with Zod (Cat 2 trust-boundary). Invalid params → `400 Bad Request`.

### `/api/time` (pivot §9.1)
5. Path: `app/api/time/route.ts`. Exports `GET` handler.
6. `export const dynamic = 'force-dynamic'`.
7. Response 200: `{ "serverTsMs": <Date.now() result> }`.
8. KV-cached 1 second via `cached('time', 'time', 1, () => ({ serverTsMs: Date.now() }))`.
9. Headers: `Cache-Control: public, max-age=1`.
10. No errors expected; if `cached` throws, route returns 500 (consistent with edge framework defaults).

### `/api/channels/[slug]/manifest` (pivot §9.3)
11. Path: `app/api/channels/[slug]/manifest/route.ts`. Exports `GET` handler.
12. Slug param validated with Zod (regex `^channel-\d+$` for D14 conformance).
13. Response 200 shape:
    ```json
    {
      "id": "<uuid>",
      "slug": "channel-1",
      "name": "<editorial name>",
      "epochAnchorMs": <ms>,
      "loopDurationSec": <int>,
      "playlist": [
        { "index": 0, "assetId": "<uuid>", "title": "...", "durationSec": <int>,
          "hlsUrl": "https://cdn.interdependent.tv/.../master.m3u8", "posterUrl": "..." }
      ]
    }
    ```
14. Drizzle query returns plain serializable shape (Cat 3 repo pattern + Cat 2 timestamp → `getTime()` conversion).
15. **`Date` columns from Drizzle MUST be converted to `*Ms` numbers** at the route boundary (Cat 2 timestamp rule). `epoch_anchor` (Postgres `timestamptz`) → `epochAnchorMs` (`number`).
16. 404 if slug not found OR `is_published = 0`.
17. KV-cached 5 minutes via `cached('manifest:<slug>', 'channel:<slug>', 300, loader)`.
18. Headers: `Cache-Control: public, max-age=300`.
19. Query function lives at `db/queries/channels.ts` as `getChannelManifestBySlug(slug)` (Cat 3 repo pattern). Route handler imports the query; never `db` directly.

## Acceptance Tests

**Edge-runtime pool tests (Cat 4 — Vitest + `@edge-runtime/vm` or Miniflare):**
- [ ] GET `/api/time` returns 200 with `{ serverTsMs: <number> }`.
- [ ] Two `/api/time` calls within 1s return identical `serverTsMs` (KV hit).
- [ ] GET `/api/channels/channel-1/manifest` returns the seeded shape from Story 1.1.
- [ ] GET `/api/channels/not-a-channel/manifest` returns 404.
- [ ] GET `/api/channels/channel-1/manifest` returns `epochAnchorMs` as a `number` (not a stringified Date).
- [ ] `purgeTag('channel:channel-1')` followed by manifest GET → cache miss → fresh DB query.

**Drizzle query tests (Cat 4 — Neon branch):**
- [ ] `getChannelManifestBySlug('channel-1')` returns the seeded channel.
- [ ] `getChannelManifestBySlug('channel-1')` returns `null` if `is_published = 0`.

## Dependencies

- **Upstream:** 1.1 (schema + seed).
- **Downstream:** 1.3 (`useServerTime` consumes `/api/time`), 1.4 (player consumes manifest).

## Risk Alignment

| ID | Risk | Mitigation in this story |
|---|---|---|
| R3 | Clock skew on viewer device | `/api/time` provides authoritative time; client offset math lives in 1.3. |

## Implementation Notes

- **No Node imports.** `Buffer`, `process.env` (use `env.<binding>` from OpenNext platform context), `fs`, `child_process` → all banned (Cat 1).
- **KV bindings.** In `wrangler.toml`, the KV binding is named `KV`. In route handlers, access via OpenNext platform context: `const { env } = getRequestContext(); const cached = await env.KV.get(...)`.
- **`/api/time` body is a single `Date.now()` call.** Keep the loader function pure; do NOT include any async DB work — it's the simplest endpoint.
- **Manifest endpoint Drizzle query:** join `channels` + `channel_assets` + `assets`; order by `channel_assets.sort_order ASC`; return the shaped object. O(1) round-trips when batched correctly.
- **Cache-Control `stale-while-revalidate`:** Optional but recommended on the manifest. Pivot §12.1 allows `max-age=30, stale-while-revalidate=300` shape; use for `/api/channels` list (Story 2.2), not strictly needed here.

## Definition of Done

- [ ] AC 1–19 met.
- [ ] Edge-runtime tests pass.
- [ ] Drizzle query tests pass against Neon branch.
- [ ] Route handlers + `lib/cache.ts` reviewed for floating-promise / `pg.Pool` / `Date.now()` violations.
- [ ] Sprint-status.yaml `1-2-api-time-and-channel-manifest-endpoints` → done.
