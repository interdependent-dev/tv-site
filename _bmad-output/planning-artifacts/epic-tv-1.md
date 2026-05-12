# Epic TV-1: Linear Channel Playback POC

| Field | Value |
|---|---|
| **Epic ID** | TV-1 |
| **Surface** | `interdependent.tv` |
| **Source spec** | `tv_poc_plan.md` v1.0 (pivot doc, Owner: Vik, 2026-05-12) |
| **Project context** | `_bmad-output/project-context.md` |
| **Status** | Pre-kickoff — §17.3 (content source) is Gated; all other §17 decisions Closed 2026-05-12 |
| **Owner** | Vik |
| **POC scale** | ≤500 concurrent viewers across ≤5 channels |
| **DoD** | All §3 acceptance gates met (G1–G5); load test passes 500-concurrent ±2s sync; risk register has no Open/High; all stories Done |

**Epic Goal.** Anonymous viewers can open `interdependent.tv`, see a "live" channel playing, switch channels via a guide, and on return either continue (VOD detour) or rejoin live. Demo-ready at 500 concurrent viewers.

**Architectural thesis.** Simulated linear TV from VOD assets on a deterministic UTC-anchored loop. No live encoding. No auth on the viewing path. LocalStorage-only viewer state. See pivot §1, §5 (ADRs), and project-context.md Categories 1 + 7.

> **Note on parser semantics:** This file uses `## Epic N:` headers as a tooling convention. Each "Epic" here is one of the **four planned sprints** under the single business-level Epic TV-1. The pre-sprint-prep bucket is "Epic 0" to capture the §17.3 gate and other floating prerequisites.

---

## Epic 0: Pre-Sprint-1 Prep (gating items)

**Goal.** Close the floating prerequisites that block Sprint 1 kickoff. None of these can be skipped or run in parallel with Sprint 1 stories.

> **Note (2026-05-12 implementation-readiness review):** Story 0.3 (infrastructure provisioning) added by party-mode consensus (unanimous). Stories 1.1, 1.2, 1.5, 4.4 all assume Cloudflare Workers preview + Neon project + DNS exist; 0.3 makes that real. Two other resolutions also recorded: D2-VOD-URL=session-canonical (Stories 3.1/3.2/3.3 updated), D3-SCHEMA=additive evolution (Stories 1.3/3.1 cross-referenced).
>
> **Note (2026-05-12 architecture review — Mux vs. Cloudflare):** Unanimous A — stay Cloudflare R2 + FFmpeg. Mux's VOD-delivery surface ≠ Mux's live-ingest surface, so Chris's prior Mux live-streaming work doesn't amortize to a delivery-layer swap. Architecture stays as designed for the 23 ready-for-dev stories.
>
> **Note (2026-05-12 analytics architecture — Plan C consensus):** Two-layer analytics: **(1)** Mux Data SDK on the R2-delivered player (Story 1.4) for delivery QoE telemetry — delivery-agnostic, free-tier, no Mux Video adoption. **(2)** Custom `playback_events` pipeline (Story 3.4) remains the authoritative creator-economy source. Cloudflare Stream Analytics evaluated via docs + free-trial memo (Story 4.7) for post-POC live-channel vendor selection — not run through POC traffic, because POC sample size cannot drive a defensible comparison. Story 0.3 updated for `MUX_DATA_KEY` provisioning.

### Story 0.1: Confirm asset inventory and close §17.3 gate

- **Status:** Draft
- **Priority:** Must (blocker)
- **Owner:** Vik (rights diligence) + Chris (content selection)
- **Decisions resolved 2026-05-12 party-mode triage:**
  - **D2:** Content source = Chris's owned MP4/MOV + pre-screening films he owns. No licensed third-party content in POC scope.
  - **D4:** Channel 1 archetype = ambient always-on / invite-to-linger.
  - **D5:** If <5 assets ready by encode-deadline → ship N (N≥3), loop to fill, document, do not relitigate scope.
  - **D6 + D7:** Rights diligence = Vik's self-attestation in writing per asset (de-risked because content is Chris-owned, not licensed).
- **AC:**
  1. Chris names 5 source assets (MP4/MOV titles + runtimes). Recorded in tracker.
  2. Per-asset Vik sign-off captured for: worldwide rights + music streaming clearance + non-SAG-AFTRA status OR documented exposure (D6/D7/D8).
  3. Each asset's "ambient-vs-flagship" classification noted; Channel 1 default channel picked.
  4. If <5 viable assets land: scope falls to N≥3 channels (per D5 default).
  5. `_bmad-output/project-context.md` §17 table updated: 17.3 moves from "Gated" to "Closed [date]."

### Story 0.2: Encoding pipeline + R2 storage setup

- **Status:** Draft
- **Priority:** Must (blocker)
- **Owner:** Chris (D13 — owns end-to-end)
- **Decisions resolved 2026-05-12 party-mode triage:**
  - **D12:** Encoding model = in-house FFmpeg (per ADR-003).
  - **D14:** Two R2 surfaces — `cdn.interdependent.tv` for HLS delivery (browser→R2 direct); viewer URLs separately on apex `interdependent.tv/channel-N`.
  - **D15:** ABR ladder = 1080p / 720p / 480p, H.264 baseline, AAC 128k stereo, 6s segments.
  - **D16:** Deadline = 1 channel HLS in R2 by Sprint 1 day 5; remaining 4 channels in R2 by Sprint 2 day 5.
  - **D17:** CORS = `Access-Control-Allow-Origin: *` for POC; explicit acceptance test added below.
  - **D18:** Source masters in private R2 bucket `tv-masters`; service-token access; path `masters/{asset_id}/source.{ext}`.
- **AC:**
  1. R2 buckets provisioned:
     - `tv-masters` (private, service-token access) — source masters live here.
     - HLS bucket (public, served via `cdn.interdependent.tv` custom domain) — HLS manifests + segments live here.
  2. FFmpeg script encodes 1 source MP4/MOV → 3-rung HLS ladder per D15 → uploads to HLS bucket. Reproducible (script lives in repo under `scripts/encode-hls.sh` or similar).
  3. Custom domain `cdn.interdependent.tv` resolves; SSL cert valid; `curl -I https://cdn.interdependent.tv/<sample-asset>/master.m3u8` returns 200.
  4. **CORS acceptance test:** Open a non-Safari browser, run a `fetch()` against `cdn.interdependent.tv` from a different origin context (DevTools console on a different page), confirm no CORS error in console. Documented as a one-step manual test in Story 0.2 close-out notes.
  5. Story 1.1's seed script has a verified path to a non-null, working `hls_url` for the first channel before Story 1.4 demo.
- **Notes:** D13 places ownership with Chris (not Vik) — the encoding pipeline is Chris's lane. Vik retains Story 0.1 (rights diligence).

### Story 0.3: Cloudflare + Neon + DNS infrastructure provisioning

- **Status:** Ready-for-dev (added 2026-05-12 implementation-readiness review, party-mode unanimous)
- **Priority:** Must (blocker)
- **Owner:** Chris (end-to-end)
- **Decisions resolved:** D1-INFRA — own provisioning as a Story rather than burying ACs in 1.1/1.2 or a README.
- **AC summary:**
  1. Cloudflare account + Workers project + `wrangler.toml` + KV namespace provisioned; preview deploy succeeds.
  2. Neon project + dev branch provisioned; `$DATABASE_URL` reachable; `neonConfig.fetchConnectionCache = true` documented.
  3. DNS: `interdependent.tv` (apex, Workers) + `cdn.interdependent.tv` (R2 HLS, cross-ref Story 0.2) both resolve with valid SSL.
  4. Secrets hygiene: `.dev.vars` gitignored; `.dev.vars.example` committed; production secrets via `wrangler secret put`.
  5. `docs/INFRASTRUCTURE.md` documents bootstrap + deploy paths.
- **Blocks:** 1.1, 1.2, 1.5, 2.5, 4.4.

---

## Epic 1: Sprint 1 — Schedule and single-channel playback

**Sprint goal.** One channel, one viewer, correct live position. Two browser windows on the same channel stay within ±2s.

### Story 1.1: Data model and seed (with schema additions from §17 resolution)

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** Story 0.1 closed, Story 0.2 in progress (at least Channel 1 encoded to HLS)
- **AC:**
  1. Drizzle schema from pivot §8.1 migrated to Neon via `drizzle-kit migrate`.
  2. **Schema additions accepted from party-mode (project-context.md Category 6):**
     - `channels.channelType` discriminator column (`'loop' | 'live'`, default `'loop'`).
     - `assets.expiresAt` (nullable timestamp with timezone).
     - `assets.territories` (nullable text array).
     - `playback_events.countryCode` (nullable text — for §17.7 geo-aware analytics).
  3. **Channel slug convention (D14 resolution):** slugs are `channel-1`, `channel-2`, ... `channel-5`. NOT `ch-cinema`-style. Editorial channel names live in `channels.name`, not the slug.
  4. Seed script inserts 1 channel (Channel 1 per D4 = ambient always-on), with assets from Story 0.2's HLS output, totaling 4–8h loop, with valid `cdn.interdependent.tv/...` `hls_url`.
  5. `loop_duration_sec = SUM(asset.duration_sec)` computed correctly by seed; invariant asserted before exit.
  6. Seed asserts every published asset has non-null `hls_url` and `duration_sec > 0`.
- **Refs:** project-context.md Cat 3 (denormalization invariant), Cat 7 #7.

### Story 1.2: API time and channel manifest endpoints

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 1.1
- **AC:**
  1. `GET /api/time` returns `{ serverTsMs }`. KV-cached 1s. `export const runtime = 'edge'` + `export const dynamic = 'force-dynamic'`.
  2. `GET /api/channels/[slug]/manifest` returns shape from pivot §9.3. 404 if slug not published.
  3. KV-cached with TTL 5min; `Cache-Control` headers explicit per §12.1.
  4. Tag-purge helper exists: `purgeTag('channel:${slug}')`.
  5. All routes export edge runtime. All inputs validated with Zod (project-context.md Cat 2 trust-boundary rule).
- **Refs:** project-context.md Cat 1 (edge runtime contract), Cat 3 (KV access patterns).

### Story 1.3: useServerTime and useSession hooks

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 1.2
- **AC:**
  1. `useServerTime` exposes `getServerNow(): EpochMs` with measured offset; initial fetch on mount; re-syncs every 5 min; re-syncs on `visibilitychange → visible` after >60s hidden.
  2. `useSession` reads/writes `itv.session.v1`; Zod-validates on read; throttled writes (1 per 2s); final write on BOTH `visibilitychange → hidden` and `beforeunload`.
  3. Unit tests for offset calculation, throttling, schema validation — all using injected clocks per project-context.md Cat 4.
- **Refs:** project-context.md Cat 3 (hook contracts), Cat 7 #2.

### Story 1.4: ChannelPlayer minimal viable

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 1.2, 1.3, 0.2 closed (CORS verified)
- **AC:**
  1. Loads manifest for `default` channel.
  2. Computes `(currentAsset, offsetSec)` via `currentProgram(nowMs, channel)` pure function — `Date.now()` BANNED inside the function.
  3. hls.js attached to `<video>` on non-Safari; native `<video>` on Safari (detected via `canPlayType('application/vnd.apple.mpegurl')`, NOT user-agent).
  4. Seek sequence enforced: hls.js path `MANIFEST_PARSED` → `currentTime =` → `play()`; Safari path `loadedmetadata` → `currentTime =` → `play()`.
  5. `lowLatencyMode: false` set in hls.js config.
  6. On `ended`: advance to `playlist[(index+1) % length]`, seek 0, play.
  7. `useEffect` cleanup: every Hls instance destroyed before reassign or unmount.
  8. Writes session to LocalStorage on `pause`, `timeupdate` (5s throttle), `beforeunload`, `visibilitychange → hidden`.
- **Acceptance demo:** Open `/` → channel auto-plays at the correct live position. Refresh after 60s → resumes within ±2s of the live position another viewer (different browser) is also seeing.
- **Refs:** project-context.md Cat 1 (player rules), Cat 7 #2, #5, #12, #13.

### Story 1.5: Cold-load performance baseline

- **Status:** Draft
- **Priority:** Should
- **Dependencies:** 1.4
- **AC:**
  1. Lighthouse measurements captured on simulated broadband.
  2. Time to first frame ≤ 2.5s p95.
  3. Manifest fetch is parallel with hls.js bundle load (no waterfall verified via DevTools waterfall trace).

**Sprint 1 exit criteria.** Default channel plays at correct live position from cold load. Two browser windows on the same channel stay within ±2s.

---

## Epic 2: Sprint 2 — Multi-channel and channel guide

**Sprint goal.** Five channels published. Viewers can switch via the guide.

### Story 2.1: Seed 4 additional channels

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 0.1 closed (full asset inventory), 1.1
- **AC:** 5 published channels, each distinct loop, loops 3h–8h, `sort_order` set. All assets have valid `hls_url`. `loop_duration_sec` invariant asserted on each.

### Story 2.2: GET /api/channels (list endpoint)

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 2.1
- **AC:**
  1. Returns pivot §9.2 shape.
  2. Computes `current` and `next` per channel via the same algorithm as `currentProgram(nowMs)`.
  3. KV-cached 30s with `stale-while-revalidate=300`.
  4. Postgres query is O(channels) — no per-asset N+1. Verified via query analysis.

### Story 2.3: ChannelGuide quick-switch row

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 2.2
- **AC:**
  1. Horizontal scrollable row of channel tiles.
  2. Each tile: logo, current program poster, title, time-remaining bar.
  3. Click → `/watch/[slug]` via `router.push` (SPA nav; no full reload).
  4. Active channel visually distinct.

### Story 2.4: /[slug] apex route + player wiring (D14 resolution: no `/watch/` prefix)

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 2.3, 1.4
- **AC:**
  1. Route at `app/(tv)/[slug]/page.tsx` renders Player for that slug. **Apex path — no `/watch/` prefix per D14.**
  2. `[slug]/page.tsx` validates the slug against the channels table at request time. Non-channel paths return Next.js `notFound()` (404 via `not-found.tsx`). Prevents catch-all collision with future top-level routes.
  3. Player tears down hls.js cleanly on slug change (verified via memory-leak test: 10 switches, one active `<video>`, zero detached `HTMLMediaElement`).
  4. Channel change → LocalStorage update.
  5. URL is shareable: opening `interdependent.tv/channel-1` from a new tab works identically.

### Story 2.5: Concurrency soak test

- **Status:** Draft
- **Priority:** Should
- **Dependencies:** 2.4
- **AC:**
  1. k6 or Artillery script simulates 200 concurrent sessions across 5 channels — running against a **deployed Workers preview env** (not localhost).
  2. Edge p95 response times: `/api/time` < 50ms, `/api/channels` < 100ms, manifest < 100ms.
  3. KV hit rate ≥ 95% after warm-up cycle.

**Sprint 2 exit criteria.** Two browser windows on different channels play simultaneously. Switching between any two channels completes in ≤500ms.

---

## Epic 3: Sprint 3 — Resume / live decision UX

**Sprint goal.** Coming back works. The resume vs. live choice is explicit and correct.

### Story 3.1: Resume modal

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 2.4
- **AC:**
  1. Mounts on `/` ONLY (not `/watch/[slug]`).
  2. Implements decision matrix from pivot §11.2 exactly — one branch per row, no improvisation.
  3. Keyboard accessible; large tap targets on mobile.
  4. Dismissable to either CTA; never re-shows in same session.
  5. Playwright test per row of the matrix (5 tests).

### Story 3.2: VOD detour mode in player

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 3.1
- **AC:**
  1. Player accepts `PlayerMode = { kind: 'vod-detour', assetId, resumeMs, returnToSlug }` per project-context.md Cat 2.
  2. Plays asset standalone with full scrubber.
  3. On `ended`: navigates to `/watch/[returnToSlug]` and resumes live.
  4. Chrome shows "VOD — returning to *Channel* live in M:SS" badge once <30s remain.
  5. User can exit detour early via "Go live now" CTA in chrome.
  6. **Edge case AC (John's flag from §17 party):** If user scrubs forward during VOD detour and remaining time drops below countdown threshold, the badge transitions correctly without flickering.

### Story 3.3: Session age and edge case handling

- **Status:** Draft
- **Priority:** Must
- **Dependencies:** 3.1
- **AC:**
  1. Session > 24h old → ignore; auto-play live on lastChannel.
  2. Last channel unpublished → fallback to default channel + show toast.
  3. Last asset deleted → modal degrades to single "Browse channels" CTA (matrix row 5).

### Story 3.4: Analytics events (designed for creator-economy signals, not ad-targeting)

- **Status:** Draft
- **Priority:** Should
- **Dependencies:** 3.1, 3.2
- **AC:**
  1. `POST /api/events` accepts batched events per pivot §9.5. **Returns 204 unconditionally** (Postgres write failure is caught + logged + swallowed). See project-context.md Cat 7 #6.
  2. Client batches up to 20 events / 5s, fire-and-forget.
  3. Event types: `start`, `pause`, `resume`, `heartbeat`, `channel_change`, `vod_detour_enter`, `vod_detour_exit`, `end`.
  4. **Each event payload includes `countryCode`** (from `request.cf.country` Cloudflare header, per project-context.md §17.7 resolution and Cat 7 #17.7-cross-ref). NOT geo-fencing — just visibility.
  5. **Analytics queries designed for creator-economy metrics** (Victor's flag): per-channel completion rate, channel loyalty (return rate), per-asset retention curves. NOT ad-targeting taxonomy.

**Sprint 3 exit criteria.** All five session-state scenarios in pivot §11.2 demoable end-to-end.

---

## Epic 4: Sprint 4 — Polish and demo readiness

**Sprint goal.** No console errors. Smooth at 500 concurrent. Demo-grade visuals.

### Story 4.1: Error states + retry

- **Status:** Draft
- **Priority:** Must
- **AC:** All states from pivot §11.5 implemented and visually polished. Retry policy: 3 attempts with exponential backoff (1s / 2s / 4s). After 3 failures, `<PlayerError>` with "Try another channel" CTA.

### Story 4.2: Loading skeletons (velvet-rope aesthetic)

- **Status:** Draft
- **Priority:** Must
- **AC:** Brand-aligned skeletons for guide and player. No flashes of unstyled content.

### Story 4.3: Mobile responsiveness pass

- **Status:** Draft
- **Priority:** Must
- **AC:**
  1. iPhone (iOS 17+) Safari: full functionality, native HLS path verified.
  2. Android Chrome: full functionality, hls.js path verified.
  3. Portrait and landscape both work.
  4. Touch-target sizes ≥ 44pt.

### Story 4.4: 500-concurrent load test + tuning

- **Status:** Draft
- **Priority:** Must
- **AC:**
  1. 500 concurrent simulated sessions for 30 min, against deployed Workers preview env.
  2. p95 cold-start ≤ 2.5s sustained.
  3. KV hit rate ≥ 98% after warm-up.
  4. Postgres CPU < 30%.
  5. R2 egress within free tier projection (extrapolated).
  6. Sprint 4 NOT declared complete until k6 passes this gate per project-context.md Cat 4.

### Story 4.5: Keyboard nav full coverage

- **Status:** Draft
- **Priority:** Should
- **AC:** Full pivot §11.6 keybinding table works (Space/K, M, F, G/↓, Esc, ↑↓, ←→, Enter, 1–9 for channel jump).

### Story 4.6: Internal demo flow walkthrough doc

- **Status:** Draft
- **Priority:** Should
- **AC:** Scripted demo flow doc covering all 5 acceptance gates (G1–G5) for external stakeholders. Authored by Sophia in collaboration with Paige (per project-context.md Cat 6).

### Story 4.7: Cloudflare Stream Analytics evaluation memo (Plan C deferred bake-off)

- **Status:** Ready-for-dev (added 2026-05-12 architecture review, Plan C consensus)
- **Priority:** Should
- **Owner:** Chris (decision-maker; product-evaluation lens, not implementation)
- **Decisions resolved:** Plan C analytics — Mux Data SDK ships in POC player (Story 1.4); Stream Analytics evaluated via docs + free-trial channel, NOT POC delivery layer. This memo is the post-POC vendor-selection input that prior agents argued the POC sample size couldn't generate defensibly.
- **AC summary:** Free-trial Stream account + one short test asset + a 2-4 page memo at `_bmad-output/planning-artifacts/stream-vs-mux-analytics-memo.md` covering: product surface comparison, creator-economy relevance (with the honest "neither answers this; Story 3.4 does"), integration coupling, production-scale pricing, and a post-POC vendor recommendation with caveats.
- **Blocks:** Nothing in this epic — informs post-POC live creator channel decision.

**Sprint 4 exit criteria.** Epic TV-1 complete. Ready for external demo.

---

## Open / deferred (post-POC)

These were surfaced by the §17 party-mode session as worth tracking but explicitly NOT in this epic's scope:

- **One true-live creator channel** (Victor's flip on 17.1) — feature addition for post-POC, requires Mux Live or Cloudflare Stream Live integration (~$3/hr ingest).
- **Passive WebAuthn enrollment at 90s of watch** (Victor's flip on 17.2) — contribution-flow enhancement; depends on auth-pivot v2 sequencing.
- **Federated creator channel from THE LOT** (Victor's flip on 17.6, rejected for POC) — revisit when first signed creator is ready to broadcast.
- **Cross-device session linkage spec** (Winston's flag) — design the anonymous-session → WebAuthn-credential bridge before auth Epic ships.
- **Analytics pipeline for creator-economy metrics** — Story 3.4 captures the events; the downstream analytics tooling (Cloudflare Analytics Engine, Tinybird, or Clickhouse) is a Scale Sprint §14.3 decision.

## Risk register (live state)

Mirrors pivot §16 with status updates:

| ID | Risk | Status | Note |
|---|---|---|---|
| R1 | Asset encoding pipeline absent | Open / High | Story 0.2 — closes pre-Sprint-1 |
| R2 | hls.js / Safari behavioral diffs | Open | Story 4.3 test matrix |
| R3 | Clock skew on viewer device | Open | Re-sync every 5 min + on drift >500ms; Story 1.3 |
| R4 | Accessibility (CC, audio desc) deferred | Open | Post-POC |
| R5 | Content licensing unclear | Open / High | Story 0.1 — closes pre-Sprint-1 |
| R6 | LocalStorage cleared | Closed | Graceful degradation in 1.3 |
| R7 | "Live" feels fake — viewers notice loop | Open | Loop ≥ 6h per channel; editorial; revisit post-demo |
| R8 | Schedule edit during heavy viewing | Closed | Documented in pivot §12.3 |
| R9 | Mobile data overage on cellular | Open | Deferred to Sprint 4 |
| R10 | Browser autoplay policies | Closed | `muted` start pattern in 1.4 |
