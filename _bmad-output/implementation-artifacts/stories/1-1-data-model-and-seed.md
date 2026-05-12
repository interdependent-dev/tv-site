---
story_id: 1.1
story_key: 1-1-data-model-and-seed
epic: 1
sprint: Sprint 1 — Schedule and single-channel playback
title: Drizzle schema migration + seed script (Channel 1)
status: ready-for-dev
priority: Must
owners:
  - Amelia (impl)
effort: M
dependencies: ["0.1 (assets confirmed)", "0.2 (Channel 1 HLS in R2)"]
blocks: ["1.2", "2.1", "2.2"]
risks: []
project_context_refs:
  - "Cat 1 (Neon HTTP driver, Drizzle pg-core, edge runtime)"
  - "Cat 3 (Drizzle query patterns, denormalization invariant)"
  - "Cat 7 #7 (loop_duration_sec invariant)"
pivot_doc_refs: ["§8.1 (schema)", "§8.3 (invariants)"]
decisions_resolved: ["D14 (slug convention)"]
---

# Story 1.1: Drizzle Schema Migration + Seed Script (Channel 1)

## User Story

As **the platform**,
I want a migrated Postgres schema and a deterministic seed script that loads Channel 1 with valid HLS assets,
So that subsequent API stories (1.2) and player stories (1.4) have a stable data foundation to read from.

## Context

The schema in pivot §8.1 is the baseline. Three schema additions accepted from the 2026-05-12 §17 party-mode resolution (project-context.md Cat 6):
- `channels.channelType` discriminator — anticipates true-live channel without future migration.
- `assets.expiresAt` + `assets.territories` — anticipates licensed content without future migration.
- `playback_events.countryCode` — geo-aware analytics per §17.7 resolution.

The slug convention is `channel-1`, `channel-2`, ..., `channel-5` (D14). Editorial channel names live in `channels.name`, not the slug.

## Acceptance Criteria

1. **Drizzle schema file** at `db/schema.ts` (or `apps/tv/db/schema.ts` if Nx multi-app) implements the 5 tables from pivot §8.1:
   - `channels` (id, slug, name, description, logo_url, epoch_anchor, loop_duration_sec, sort_order, is_published, **channelType** [new], created_at, updated_at)
   - `assets` (id, title, description, duration_sec, hls_url, poster_url, license_state, **expires_at** [new], **territories** [new], created_at)
   - `channel_assets` (channel_id, asset_id, sort_order)
   - `schedule_entries` (reserved; not used in seed)
   - `playback_events` (id BIGINT, session_id, channel_id, asset_id, event_type, position_sec, **country_code** [new], ts)
2. **`channels.channelType` column** is `text` with values `'loop' | 'live'`, default `'loop'`. CHECK constraint or app-code enforcement.
3. **`assets.territories` column** is `text[]` nullable. `assets.expiresAt` is `timestamp with time zone` nullable.
4. **`playback_events.countryCode` column** is `text` nullable, max length 2 (ISO-3166 alpha-2).
5. **Indexes per pivot §8.1** are created:
   - `channels_published_idx ON (is_published, sort_order)`
   - `channel_assets_channel_idx ON (channel_id, sort_order)`
   - `events_session_idx ON (session_id, ts)`
   - `events_channel_idx ON (channel_id, ts)`
6. **Migration applied** to a Neon branch via `pnpm drizzle-kit migrate` (NEVER `push` against prod — project-context.md Cat 1).
7. **Seed script** at `db/seed.ts` (or similar) is idempotent and inserts:
   - 1 channel with `slug = 'channel-1'`, `channelType = 'loop'`, `is_published = 1`, `sort_order = 0`, ambient-always-on archetype (per D4).
   - Assets from Story 0.2's encoded HLS output (Channel 1's loop). Each asset has non-null `hls_url` pointing to `cdn.interdependent.tv/{asset-uuid}/master.m3u8`.
   - Channel-asset linkage rows in `channel_assets` with `sort_order = 0, 1, 2, ...`.
8. **Invariants asserted before exit:**
   - `loop_duration_sec === SUM(asset.duration_sec)` for the channel. Throws on mismatch.
   - Every asset referenced by the channel has non-null `hls_url`.
   - Every asset has `duration_sec > 0`.
9. **`epoch_anchor`** set to a fixed UTC timestamp (e.g., `2026-01-01T00:00:00Z`) — identical across viewers per pivot §7.1.
10. **Seed script is idempotent.** Running twice produces the same database state (no duplicate rows).

## Acceptance Tests

**Unit tests:**
- [ ] `loop_duration_sec` invariant test: pass a mock channel + asset list; assert recompute matches SUM.
- [ ] Slug-format test: assert `channel-N` pattern enforced (regex check in seed).

**Integration tests (against Neon branch):**
- [ ] After migration, run `\d channels` (or Drizzle introspect) — schema matches AC 1–5.
- [ ] After seed, `SELECT * FROM channels WHERE slug = 'channel-1'` returns one row with `is_published = 1`.
- [ ] After seed, `SELECT COUNT(*) FROM channel_assets WHERE channel_id = ?` matches the asset count.
- [ ] Run seed twice; assert no duplicate channels.

## Dependencies

- **Upstream:** 0.1 (asset list), 0.2 (Channel 1 encoded + in R2).
- **Downstream:** 1.2 (API needs schema), 2.1 (seed additional channels), 2.2 (channels list API).

## Risk Alignment (pivot §16)

| ID | Risk | Relevance |
|---|---|---|
| R7 | "Live" feels fake — viewers notice the loop | Loop length ≥ 6h target; seed enforces by selecting enough assets per channel. If asset runtime sums < 6h, log a TODO per D9 default (acceptable for POC). |

## Implementation Notes

- **Drizzle adapter:** `drizzle-orm/neon-http` only (project-context.md Cat 1). Wrong import (`node-postgres`) compiles locally and fails at runtime in Workers.
- **`neonConfig.fetchConnectionCache = true`** must be set in the Neon client config.
- **Migrations run OUTSIDE the Worker.** The seed script is a Node script (TypeScript via `tsx` or `bun run`); never imported by an edge route handler.
- **`channelType` enum-like column.** Use `text` + CHECK constraint OR Drizzle's `pgEnum`. Decision: use `pgEnum` for type safety, named `channel_type_enum`.
- **`epoch_anchor` value:** Pick `2026-01-01T00:00:00Z` for POC. Any earlier fixed UTC is fine; the value is arbitrary but must be stable.
- **Schema file naming convention:** Drizzle tables use `pgTable("channels", { ... })` — column keys are camelCase in TS, snake_case in SQL (Drizzle auto-maps).

## Definition of Done

- [ ] AC 1–10 met.
- [ ] Unit + integration tests green.
- [ ] Neon branch has applied migration + seed.
- [ ] Story 1.2 unblocked.
- [ ] Sprint-status.yaml `1-1-data-model-and-seed` → done.
