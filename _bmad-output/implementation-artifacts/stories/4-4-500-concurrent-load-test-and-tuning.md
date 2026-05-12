---
story_id: 4.4
story_key: 4-4-500-concurrent-load-test-and-tuning
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: 500-concurrent load test + tuning (G3 acceptance gate)
status: ready-for-dev
priority: Must
owners: ["Murat (test design) + Amelia (impl tuning)"]
effort: M
dependencies: ["2.5", "all prior Sprint stories"]
blocks: []
risks: []
project_context_refs:
  - "Cat 4 (k6 against deployed Workers, warm-up cycle, acceptance gate)"
pivot_doc_refs: ["§3 G3 (500 concurrent, ±2s sync, p95 ≤2.5s)", "Sprint 4 Story 4.4"]
---

# Story 4.4: 500-Concurrent Load Test + Tuning

## User Story

As **the platform engineering team**,
I want a 500-concurrent load test against a deployed Workers preview that meets all G3 gates,
So that Epic TV-1 is empirically demo-ready, not aspirationally so.

**This is the load gate that determines Sprint 4 exit AND Epic TV-1 closure.**

## Context

Sprint 2 Story 2.5 was the 200-concurrent dry run. This story scales to 500 (the full pivot G3 gate) and adds tuning as needed.

Per project-context.md Cat 4:
- k6 MUST target deployed Workers preview env. NEVER localhost.
- Warm-up cycle is mandatory.
- Tag-purge integration test must already exist (added during 1.2 or earlier).

## Acceptance Criteria

1. **k6 script** at `tests/load/sprint-4-500-concurrent.js`. Extension of `sprint-2-200-concurrent.js` with higher VU count.
2. **Profile:** 500 VUs, distributed ~100 per channel across the 5 channels. Ramp 0→500 over 60s; hold 500 for **30 minutes** (long enough to expose memory leaks, KV miss storms, Postgres connection saturation).
3. **Warm-up:** one request per channel + per endpoint to pre-populate KV. Documented in test header.
4. **Acceptance gates (pivot §3 G3 + Sprint 4 Story 4.4):**
   - p95 cold-start ≤ 2.5s sustained.
   - KV hit rate ≥ 98% after warm-up.
   - Postgres CPU < 30%.
   - R2 egress within free-tier projection (extrapolated; ≤10TB/month projected for sustained 500 concurrent).
   - Zero 5xx during 30-min steady state.
   - **±2s in-channel sync** measured by spawning two Playwright contexts during the load run and verifying their `video.currentTime` divergence.
5. **Tuning** as needed:
   - If p95 fails → identify dominant cost via Workers tail + Lighthouse; file follow-up story OR adjust in-line.
   - If KV hit rate < 98% → audit cache helper for misses; possibly extend TTLs (within pivot §12.1 envelope).
   - If Postgres CPU > 30% → review N+1 queries; add connection pooling if needed; consider Neon Scale tier.
6. **Report** at `_bmad-output/implementation-artifacts/perf/sprint-4-500-concurrent.md`:
   - Test profile + target URL.
   - All metrics with pass/fail flags.
   - Tuning changes applied during the story (if any).
   - Recommendations for Scale Sprint §14 transitions.
7. **Sprint 4 NOT declared complete** until k6 passes this gate against a deployed preview.

## Acceptance Tests

- [ ] `k6 run tests/load/sprint-4-500-concurrent.js` completes without errors over 30 min sustained.
- [ ] All gates green per AC 4.
- [ ] Playwright sync test green during k6 run (manually orchestrated for the POC; CI integration is post-POC).

## Dependencies

- **Upstream:** All Sprint 1–3 stories (the system needs to be functional end-to-end).
- **Downstream:** none — this is the gate. Story 4.6 (demo doc) follows once green.

## Risk Alignment

(No direct §16 risks; this story IS the empirical validation of the architecture.)

## Implementation Notes

- **Warm-up cycle (Cat 4 rule):** before VU ramp, run a single-VU pass that hits each of: `/api/time`, `/api/channels`, `/api/channels/channel-1/manifest` (through channel-5). Confirms KV is populated.
- **k6 stages:**
  ```js
  stages: [
    { duration: '60s', target: 500 },
    { duration: '30m', target: 500 },
    { duration: '60s', target: 0 },
  ]
  ```
- **Sustained 30-min runtime cost:** Workers Free is 100k req/day. At 500 VUs * 1 batched req per 30s = ~17 req/sec = ~60k req/hour. 30 min run = 30k req. Single run fits Free; daily repeats need Paid tier ($5/mo).
- **Postgres CPU monitoring:** Neon dashboard exposes CPU metrics. Capture screenshot or use Neon API for the run window.
- **Sync test orchestration:** Spawn 2 Playwright browser contexts on the same channel while k6 ramps. Sample `video.currentTime` every 5s for 5 min. Assert max divergence ≤ 2s.

## Definition of Done

- [ ] AC 1–7 met.
- [ ] All G3 gates GREEN.
- [ ] Report committed.
- [ ] Sprint-status.yaml `4-4-500-concurrent-load-test-and-tuning` → done.
- [ ] Epic TV-1 DoD criterion (load test 500 concurrent) satisfied.
