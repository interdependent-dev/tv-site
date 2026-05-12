---
story_id: 2.5
story_key: 2-5-concurrency-soak-test
epic: 2
sprint: Sprint 2 — Multi-channel and channel guide
title: 200-concurrent soak test (k6) against deployed Workers preview
status: ready-for-dev
priority: Should
owners: ["Murat (test design) + Amelia (impl)"]
effort: M
dependencies: ["2.4"]
blocks: []
risks: []
project_context_refs:
  - "Cat 4 (k6 against deployed Workers, NOT localhost)"
  - "Cat 7 #16 (bundle size as correctness)"
pivot_doc_refs: ["Sprint 2 Story 2.5", "§3 G3 (500 concurrent, ±2s sync)"]
---

# Story 2.5: 200-Concurrent Soak Test

## User Story

As **the platform engineering team**,
I want a load test showing 200 concurrent sessions across 5 channels with healthy edge latencies and high KV hit rate,
So that we have empirical confidence in the architecture before Sprint 4 pushes to 500 concurrent (the full G3 gate).

## Context

Sprint 2's concurrency target is 200 — half of the full G3 (500 concurrent at Sprint 4). This story establishes the test harness, validates the architecture at a non-trivial scale, and exposes any hot-path bottlenecks before they compound at 500.

**Cat 4 rule: k6 MUST run against a deployed Cloudflare Workers preview env.** `next dev` does NOT emulate Workers KV TTL, edge runtime constraints, or cold-start behavior. Loading localhost is meaningless.

## Acceptance Criteria

1. **k6 script** at `tests/load/sprint-2-200-concurrent.js`. Versioned in repo.
2. **Target:** a deployed Workers preview env (`wrangler deploy --env staging` or Cloudflare Pages preview branch). Documented in test header comment.
3. **Test profile:** 200 virtual users (VUs) across 5 channels, distributed roughly 40 VUs per channel. Ramp: 0 → 200 over 30s; hold 200 for 5 min; ramp down over 30s.
4. **Each VU lifecycle simulates a viewer:**
   - GET `/api/channels` once on session start.
   - GET `/api/channels/{slug}/manifest` once.
   - GET `/api/time` every 5 minutes (per `useServerTime` re-sync cadence).
   - GET HLS segments from `cdn.interdependent.tv` simulated at 1 segment per 6s (with one rung of the ABR ladder).
   - Optional: a small fraction (10%) does a channel switch mid-session.
5. **Warm-up:** before measurement window, one request per channel hits the Workers preview to populate KV. Skipping warm-up produces misleading first-30s metrics dominated by KV-miss cold paths.
6. **Acceptance gates (pivot Sprint 2 Story 2.5):**
   - Edge p95 `/api/time` < 50ms.
   - Edge p95 `/api/channels` < 100ms.
   - Edge p95 manifest < 100ms.
   - KV hit rate ≥ 95% post-warm-up.
   - No 5xx responses during the 5-min steady state.
7. **Report** at `_bmad-output/implementation-artifacts/perf/sprint-2-soak.md`:
   - Test profile + target URL.
   - p50, p95, p99 per endpoint.
   - KV hit-rate metric.
   - Workers CPU time (from Cloudflare dashboard or wrangler tail).
   - Pass/fail vs. gates.

## Acceptance Tests

- [ ] `k6 run tests/load/sprint-2-200-concurrent.js` completes without errors.
- [ ] All gates green per AC 6.
- [ ] If any gate fails, file a tuning story BEFORE Sprint 2 exit.

## Dependencies

- **Upstream:** 2.4 (multi-channel + slug routing live).
- **Downstream:** Sprint 4 Story 4.4 (extends to 500 concurrent).

## Risk Alignment

| ID | Risk | Mitigation |
|---|---|---|
| R3 | Clock skew on viewer device | KV-cached `/api/time` at 1s ensures consistent server clock under load. |
| R8 | Schedule edit during heavy viewing | Out of scope for this test — no edits during the 5-min window. |

## Implementation Notes

- **k6 script structure:** Use `k6/http` for the JSON endpoints. HLS segment fetch can be simulated with a sub-request to one segment URL per loop iteration (no need to actually decode video — we're measuring CDN + Workers, not the player).
- **Throughput math sanity check:** 200 VUs × 1 manifest req per 5 min = 0.67 req/sec on manifest. KV TTL is 5 min → essentially 100% hit rate after warm-up. The hot path is `/api/time` (200 VUs × 1 req per 5 min = 0.67 req/sec, KV-cached 1s — still mostly hits). Real load comes from HLS segments hitting R2 — that's CDN, not Workers.
- **CPU time on Workers:** monitor `wrangler tail` during the run. Workers Free has CPU time limits (10ms / 50ms paid). Watch for spikes near the limit.
- **Cost:** Workers Free is 100k req/day; at 0.67 req/sec sustained = 2400 req/hour. 5 min run = 200 req. Well within free tier. R2 is the cost center if HLS segments are fetched at scale — but with KV-cached manifests and segments served direct from R2 edge (Cloudflare's CDN layer in front of R2), it's negligible.

## Definition of Done

- [ ] AC 1–7 met.
- [ ] Report committed.
- [ ] All Sprint 2 gates green OR follow-up tuning story filed.
- [ ] Sprint-status.yaml `2-5-concurrency-soak-test` → done.
