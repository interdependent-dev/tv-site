---
story_id: 4.7
story_key: 4-7-stream-analytics-evaluation-memo
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Cloudflare Stream Analytics evaluation memo (Plan C deferred bake-off)
status: ready-for-dev
priority: Should
owners:
  - Chris (decision-maker; product-evaluation lens)
effort: S
dependencies: []
blocks: []
risks: []
project_context_refs:
  - "Cat 6 (documentation discipline — research memo, not narrative)"
pivot_doc_refs: ["§14 scale path", "Open / deferred post-POC: 'one true-live creator channel'"]
decisions_resolved: ["Plan C analytics (2026-05-12 party — Mux Data SDK ships in POC; Stream Analytics evaluated via docs + free-trial, NOT POC delivery layer)"]
---

# Story 4.7: Cloudflare Stream Analytics Evaluation Memo

## User Story

As **Chris (post-POC vendor-selection decision-maker)**,
I want a short written evaluation of Cloudflare Stream Analytics vs. Mux Data — based on docs + a free-trial channel — captured before Epic TV-1 demo,
So that when the post-POC live creator channel decision arrives, I have a documented comparison from a sober-mind moment rather than improvising under launch pressure.

## Context

The 2026-05-12 party (Plan C consensus) resolved that running Cloudflare Stream as a delivery layer for one POC channel — purely to populate its Analytics dashboard — was not worth the architectural cost. The agents agreed on three grounds:
1. **Methodological:** POC sample size (LP demos + internal + small creator cohort) cannot generate statistically meaningful comparison data. The dashboards would render but not be defensibly comparable.
2. **Confounding:** Channels differ in content, traffic weighting, and viewer overlap. Attribution of dashboard differences to *vendor* is invalid.
3. **Strategic:** Both Mux Data and Stream Analytics are QoE telemetry products. The creator-economy thesis lives in Story 3.4's `playback_events` pipeline, not in either vendor's dashboard.

But the underlying decision — Mux Live vs. Cloudflare Stream Live for the post-POC live creator channel — is real and remains open. This story produces the artifact that informs that future decision **without compromising the POC architecture.**

## Acceptance Criteria

### Setup

1. **Free-trial Cloudflare Stream account** provisioned (separate from POC's main Cloudflare account, or a dedicated Stream subscription). Free trial offers ~5 GB storage + ~10K minutes delivered.
2. **One short test asset** (1-5 min sample MP4/MOV — can reuse one of Chris's owned assets from Story 0.1) uploaded to Stream via the dashboard or Stream API.
3. **One test session** played through end-to-end via the Stream-provided player OR a custom hls.js player pointed at the Stream HLS URL. Single session, no production traffic — just exercising the dashboard.

### Evaluation (the memo itself)

4. **Memo lives at `_bmad-output/planning-artifacts/stream-vs-mux-analytics-memo.md`.** Markdown, 2-4 pages.
5. **Section 1 — Product surface comparison:**
   - **Dashboard organization:** what's the default view? How is data sliced (per-asset, per-viewer, geo, time-of-day)?
   - **QoE metrics available:** startup time, rebuffer ratio, playback failure rate, exits before video starts, video startup failure. Note which platform exposes each, in what granularity.
   - **Custom dimensions / metadata:** can you attach `viewer_user_id`, `video_title`, `video_series`, custom fields? Both platforms support this — note depth and ergonomics.
   - **Real-time vs. batched:** how fresh is the data? Mux Data is near-real-time. Stream's freshness — verify.
   - **API access:** can you query metrics programmatically for custom dashboards or BI? Both have APIs — compare ergonomics.
6. **Section 2 — Creator-economy relevance:**
   - Note which platform (if either) exposes metrics directly relevant to creator-economy thesis: completion rate per asset (vs. just "watch time"), return-visitor cohorts, channel-loyalty patterns, time-of-day engagement curves.
   - **Honest answer expected:** neither platform's native dashboard is built for creator-economy attribution. Both are QoE products. The custom `playback_events` pipeline (Story 3.4) is the creator-economy authoritative surface regardless. The memo should say this plainly.
7. **Section 3 — Integration coupling:**
   - Mux Data: delivery-agnostic SDK. Story 1.4 already ships it. Zero extra delivery commitment.
   - Stream Analytics: requires Stream as the delivery layer. Adoption means migrating delivery off R2 for at least some channels, with the schema-leakage / D14-breakage costs the prior party already weighed.
8. **Section 4 — Pricing comparison at production scale (post-POC live channel):**
   - Mux Data: free tier 100K plays/month; paid scales with views. Cite current pricing page snapshot.
   - Cloudflare Stream: $5/1000 min stored + $1/1000 min delivered. Analytics included.
   - Project costs for a hypothetical 10K-MAU creator channel running 4 hrs/day on each platform. Order-of-magnitude only.
9. **Section 5 — Recommendation:**
   - For the live creator channel post-POC: which vendor, with what caveats? Include the "no decision yet" option explicitly — recommend revisiting after the first live creator is identified, with their specific use case as the deciding factor.
   - Note any blocker the memo cannot resolve (e.g., "Mux Live + Stream Live latency claims need real measurement before commitment").

### Verification

10. **Memo reviewed by Chris.** Single-author artifact (no formal review process). Signed off as "done" when Chris reads it back and confirms it captures the comparison faithfully.

## Acceptance Tests / Verification

**Manual:**
- [ ] Cloudflare Stream free trial provisioned; one asset uploaded.
- [ ] Memo exists at the expected path with all 5 sections.
- [ ] Pricing claims sourced from live pricing pages (with snapshot date noted).
- [ ] Chris read-back confirms the comparison is fair.

## Dependencies

- **Upstream:** None — runs parallel to all dev work. Can be authored in any 1-2 hour Chris window across Sprints 1-4.
- **Downstream:** Post-POC vendor decision for live creator channel (out of scope for this epic).

## Risk Alignment

(No §16 risk; product-evaluation artifact.)

## Implementation Notes

- **Why Sprint 4, not Sprint 0?** This is a research task with no dev dependency. Slot it in any week before the demo. Sprint 4 placement is administrative — keeps it grouped with other "wrap up" deliverables.
- **Why Chris owns, not Amelia?** This is a decision-maker artifact, not implementation. Amelia could write it but Chris's judgment is the load-bearing input.
- **Don't over-engineer the memo.** 2-4 pages. Charts where useful. No need for statistical tests or A/B math — the methodological argument for skipping a real A/B test is itself in this story's Context section.
- **Cross-link to Story 3.4** — the memo must explicitly state that creator-economy metrics are not the QoE vendor's job, regardless of which vendor is picked.

## Definition of Done

- [ ] AC 1–10 met.
- [ ] Memo committed at `_bmad-output/planning-artifacts/stream-vs-mux-analytics-memo.md`.
- [ ] Sprint-status.yaml `4-7-stream-analytics-evaluation-memo` → done.
- [ ] Epic TV-1 DoD criterion not affected; this story is "Should" priority, not "Must."
