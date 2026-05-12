---
story_id: 2.1
story_key: 2-1-seed-4-additional-channels
epic: 2
sprint: Sprint 2 — Multi-channel and channel guide
title: Seed 4 additional channels (channel-2 through channel-5)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl) + Chris (encoding)"]
effort: S
dependencies: ["0.1 (full asset inventory)", "0.2 (Sprint 2 day 5 deadline — 4 more channels in R2)", "1.1"]
blocks: ["2.2", "2.3", "2.5"]
risks: ["R7"]
project_context_refs:
  - "Cat 3 (denormalization invariant)"
  - "Cat 7 #7 (loop_duration_sec)"
pivot_doc_refs: ["Sprint 2 Story 2.1", "§17.6 (internal-only, 5 channels)"]
---

# Story 2.1: Seed 4 Additional Channels

## User Story

As **the channel guide (Story 2.3)** and **the channel-switch flow (Story 2.4)**,
I want 4 additional published channels in the database (channel-2 through channel-5),
So that the multi-channel UX can be built and tested against real, distinct loops.

## Context

Story 1.1 seeded Channel 1 only. Sprint 2 needs the remaining 4. D16 deadline: 4 more HLS-encoded channels in R2 by Sprint 2 day 5. Story 0.2 owns the encoding; this story extends the seed script.

§17.6 settled: internal-only 5 channels. NO federated creator channel (Victor's flip was rejected, revisit post-POC). Slugs are `channel-2`, `channel-3`, `channel-4`, `channel-5` per D14.

## Acceptance Criteria

1. **Seed script extended** at `db/seed.ts` (or wherever 1.1 placed it). Inserts channels 2–5 idempotently.
2. **Each channel has:**
   - `slug` matching `^channel-[2-5]$`.
   - `name` (editorial — placeholder names per D3 default acceptable).
   - `channelType = 'loop'`.
   - `is_published = 1`.
   - `sort_order` ∈ {1, 2, 3, 4} (Channel 1 occupies `sort_order = 0`).
   - `epoch_anchor` identical to Channel 1's anchor (single shared UTC anchor for the loop math; ensures channel-switching arithmetic is consistent).
3. **Each channel's assets** are inserted into `assets` table + linked via `channel_assets` rows with sequential `sort_order`.
4. **Each loop is 3–8 hours** (pivot Sprint 2 Story 2.1 AC). Loop length per channel allowed to vary; no strict 6h+ minimum (R7 mitigation is loop-content variety, not loop length per channel).
5. **`loop_duration_sec` invariant asserted per channel** before seed exit.
6. **All HLS URLs reachable.** Seed script `HEAD`s each `hls_url` and asserts 200 before completing. If any URL fails, seed exits non-zero with a clear error naming the offending asset.
7. **Idempotency:** running the seed twice produces the same DB state.

## Acceptance Tests

- [ ] Post-seed `SELECT slug, is_published, sort_order, loop_duration_sec FROM channels ORDER BY sort_order` returns 5 rows with matching expected values.
- [ ] Post-seed `SELECT COUNT(*) FROM channel_assets WHERE channel_id IN (...)` matches sum of asset counts per channel.
- [ ] Per-channel `loop_duration_sec === SUM(asset.duration_sec)` invariant holds (assert in seed; re-verify in test).
- [ ] HEAD-reach test: every `hls_url` returns 200 from a cold curl.

## Dependencies

- **Upstream:** 0.1 (all 5 assets/channels editorial-confirmed), 0.2 day-of-Sprint-2-day-5 (channels 2–5 HLS in R2), 1.1 (schema).
- **Downstream:** 2.2 (channels list API needs >1 published channel), 2.3 (guide renders multi), 2.5 (concurrency test needs 5 channels).

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| R7 | "Live" feels fake — viewers notice the loop | Distinct programming per channel; loop variety > loop length. Editorial decision per channel name + content mix. |
| R1 | Asset encoding pipeline absent | Story 0.2 day-of-Sprint-2-day-5 deadline IS the unblock. This story's HEAD check catches missing uploads loudly. |

## Implementation Notes

- **Shared `epoch_anchor`** matters for channel-switching latency: when switching channels, all viewers compute positions against the same wall-clock origin → no anchor-drift surprises across channels.
- **`channels.description` and `channels.logo_url`** can be placeholders / empty for POC. Sprint 4 Story 4.2 (loading skeletons + brand polish) is where editorial fills these in.
- **Sequential `sort_order`** is editorial: `channel-1` (ambient, the default) at 0; the rest 1–4 in whatever order Chris prefers.

## Definition of Done

- [ ] AC 1–7 met.
- [ ] 5 channels published with valid HLS.
- [ ] HEAD-reach test green for every asset.
- [ ] Sprint-status.yaml `2-1-seed-4-additional-channels` → done.
