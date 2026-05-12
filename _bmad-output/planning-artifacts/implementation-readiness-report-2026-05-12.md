---
stepsCompleted:
  - step-01-document-discovery
  - step-02-prd-analysis (consolidated — no standalone PRD)
  - step-03-ux-analysis (consolidated — UX in pivot §11)
  - step-04-architecture-analysis (consolidated — ADRs in pivot §5 + project-context.md Cat 1-7)
  - step-05-epics-and-stories-deep-validation
  - step-06-blocker-resolution-via-party-mode
  - step-07-final-readiness-report
filesIncluded:
  - _bmad-output/project-context.md
  - _bmad-output/planning-artifacts/epic-tv-1.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
  - _bmad-output/implementation-artifacts/stories/*.md (23 stories)
---

# Implementation Readiness Assessment Report

**Date:** 2026-05-12
**Project:** tv (Epic TV-1 — Linear Channel Playback POC, surface `interdependent.tv`)
**Assessor:** Implementation-Readiness skill (Chris invoked; autonomous-mode per directive)

---

## Executive Summary

| Question | Answer |
|---|---|
| Is the planning ready for development to start? | **YES — Epic 0 prep work pending, but all stories are dev-ready in spec terms.** |
| How many stories are ready-for-dev? | **23 of 23** (22 original + 1 added during this review). |
| Critical blockers found during review? | **3 critical, 2 fix-and-go. All resolved.** |
| Are real-world prerequisites still pending? | **Yes — Story 0.1 (asset inventory) and Story 0.3 (infrastructure provisioning) need Chris/Vik execution before Sprint 1 dev work.** Specs are complete; the work itself is real-world (rights diligence, account provisioning), not story-authoring. |
| Can Sprint 1 start tomorrow? | **Only after 0.1 + 0.2 + 0.3 close. They are designed to run in parallel within Pre-Sprint-1 Prep and require Chris (encoding + accounts) + Vik (rights).** |

---

## Document Inventory

Standard BMAD planning artifacts (PRD / UX / Architecture as separate files) were **not produced** — this is intentional and appropriate for the project's Level. Planning is consolidated into:

| Artifact | Path | Role |
|---|---|---|
| Project context | `_bmad-output/project-context.md` | Source of architectural rules (Categories 1–7), §17 decision table, hook contracts, type discipline, test architecture. The de facto PRD + Architecture + UX bundle. |
| Pivot doc (referenced) | `tv_poc_plan.md` v1.0 (referenced in epic) | ADRs, §3 acceptance gates G1–G5, §11.x UX matrices, §16 risk register, §17 decisions. |
| Epic | `_bmad-output/planning-artifacts/epic-tv-1.md` | One business epic decomposed into 5 sprints (Epic 0 prep + Sprints 1–4). Live risk register mirror. |
| Sprint status | `_bmad-output/implementation-artifacts/sprint-status.yaml` | All 23 stories tracked; gating items called out. |
| Stories (23) | `_bmad-output/implementation-artifacts/stories/*.md` | Sprint 0 (3), 1 (5), 2 (5), 3 (4), 4 (6). |

**No duplicates** — single canonical location for each artifact. **No missing required documents** — implicit consolidation into project-context.md is verified consistent across story refs.

---

## Validation Findings (Original 22 Stories)

### What's exceptional

1. **Cross-referencing depth.** Every story names project-context.md Categories + Cat 7 numbered rules + pivot doc sections. Trace from AC → architectural rule → risk register is intact end-to-end.
2. **Test discipline.** Every story specifies test type (unit jsdom / edge-runtime pool / Playwright browser-mode / k6) and points to Cat 4. Murat-level rigor baked in upstream.
3. **Risk register coupling.** Stories close R# items explicitly (e.g., 0.1 closes R5, 0.2 closes R1, 4.3 closes R2). Open/Closed transitions are story-owned.
4. **POC scope discipline.** Stories defer post-POC items explicitly (DRM, accessibility, federated creator channel, EPG grid, mobile data warning). No scope creep.
5. **§17 party-mode decisions traced into ACs.** D2 (content source), D4 (Channel 1 archetype), D12–D19 (encoding pipeline + R2), D14 (slug convention), §17.7 (countryCode) all flow into specific ACs.

### What needed fixing (now resolved)

| # | Issue | Severity | Resolution |
|---|---|---|---|
| 1 | **No story owns infrastructure provisioning.** Stories 1.1, 1.2, 1.5, 2.5, 4.4 all assume Cloudflare Workers preview + Neon + DNS exist; no story creates them. | **CRITICAL** | New **Story 0.3** authored (`0-3-infrastructure-provisioning.md`). Owner: Chris. 20 ACs covering account, Workers project + `wrangler.toml`, KV namespace, Neon project + dev branch, DNS apex + cdn subdomain, secrets hygiene, `docs/INFRASTRUCTURE.md`. Blocks 1.1, 1.2, 1.5, 2.5, 4.4. |
| 2 | **VOD detour `hls_url` source ambiguous.** Story 3.2 AC 4 said "new endpoint OR manifest playlist"; 3.1 implementation notes hinted at session cache; 3.3 AC 5 said "possibly cached in session." Three plausible sources. | **CRITICAL** | Stories 3.1, 3.2, 3.3, 1.4, 2.4 all updated. **D2-VOD-URL = B (session canonical).** `session.lastAssetHlsUrl` + `session.lastAssetTitle` cached at exit time by Story 1.4 / 2.4's `updateSession` calls. Resume Modal reads them and passes them as PlayerMode props. No `/api/assets/{id}` endpoint in POC. Row 5 detection simplified to "session.lastAssetHlsUrl is null." |
| 3 | **SessionV1 schema extensions scattered.** Story 1.3 defines SessionV1; Stories 3.1 + 3.3 add `lastAssetTitle`, `resumeModalShown`, `lastAssetHlsUrl` without consolidation. | **HIGH** | **D3-SCHEMA = B (additive evolution).** Story 1.3 implementation notes now include a forward-compat clause: either declare future fields as `z.string().nullable().optional()` in 1.3, OR apply `.passthrough()`. Story 3.1 owns formal schema extension. 1.3's tests still validate launch fields strictly. |
| 4 | **Story `status: draft` vs sprint-status.yaml `ready-for-dev`** mismatch in all story frontmatter. | **HIGH** | All 23 story files now have `status: ready-for-dev` (sed-applied). Verified across all files. |
| 5 | **Story 2.4 implementation-note typo:** `ChannelPage` returned `<ChannelPage>` (self-recursion). | **MEDIUM** | Snippet rewritten to return `<ChannelPlayer slug={...} initialManifest={...} />` + `<ChannelGuide />`. |

---

## Party-Mode Decisions (2026-05-12 readiness review)

Three blockers required team consensus; all resolved via 6-agent party-mode roundtable.

| ID | Decision | Vote | Chosen Option | Rationale |
|---|---|---|---|---|
| **D1-INFRA** | Where does Cloudflare/Neon/DNS provisioning live? | **6/6 unanimous** | **A: New Story 0.3 before Sprint 1** | All voices aligned: visible, traceable, single owner, prevents silent-failure-during-test-run. Pivot consideration: pre-sprint prep is the appropriate scope for one-time setup. |
| **D2-VOD-URL** | What's the source of truth for the VOD detour `hls_url`? | **3-3 split → architect-deciding vote** | **B: Session canonical** (cache hls_url + title at exit) | Winston (A→B tie-break vote), Sally, John argued POC scope discipline beats future-proofing. Amelia, Murat, Mary preferred a dedicated endpoint (C). Architect's vote decides architectural ties per BMAD convention; POC scope discipline reinforced by §17 internal-only resolution. |
| **D3-SCHEMA** | Where to organize SessionV1 schema extensions? | **4-2** | **B: Additive evolution in 3.1/3.3** | Sally, Amelia, John, Mary chose B (right-sized stories, schemas evolve where they're consumed). Winston, Murat preferred a canonical schema file (C). Mary's caveat — write an epic-close Schema Changelog — adopted as a non-blocking enhancement; suggested artifact: `_bmad-output/planning-artifacts/session-schema-changelog.md` at Epic TV-1 close. |

Full agent transcripts available in conversation history; decisions traced into story frontmatter and inline AC comments.

---

## Story Inventory (23 total)

### Epic 0: Pre-Sprint-1 Prep (3 stories — all gating)

| Story | Title | Owner | Effort | Blocks |
|---|---|---|---|---|
| 0.1 | Confirm asset inventory + close §17.3 gate | Vik + Chris | S | 1.1, 2.1 |
| 0.2 | Encoding pipeline + R2 storage | Chris | M | 1.1, 1.4 |
| 0.3 | Cloudflare + Neon + DNS provisioning | Chris | M | 1.1, 1.2, 1.5, 2.5, 4.4 |

### Epic 1 (Sprint 1): Schedule + single-channel playback (5 stories)

| Story | Title | Owner | Effort | Dependencies |
|---|---|---|---|---|
| 1.1 | Data model + seed | Amelia | M | 0.1, 0.2, 0.3 |
| 1.2 | `/api/time` + `/api/channels/[slug]/manifest` | Amelia | M | 1.1 |
| 1.3 | `useServerTime` + `useSession` hooks | Amelia | M | 1.2 |
| 1.4 | `<ChannelPlayer>` minimal viable | Amelia | L | 1.2, 1.3, 0.2 |
| 1.5 | Cold-load performance baseline (G1 gate) | Amelia | S | 1.4, 0.3 |

### Epic 2 (Sprint 2): Multi-channel + guide (5 stories)

| Story | Title | Owner | Effort | Dependencies |
|---|---|---|---|---|
| 2.1 | Seed 4 additional channels | Amelia + Chris | S | 0.1, 0.2, 1.1 |
| 2.2 | `GET /api/channels` (list) | Amelia | M | 2.1 |
| 2.3 | `<ChannelGuide>` quick-switch row | Amelia + Sally | L | 2.2 |
| 2.4 | `/[slug]` apex route + player wiring | Amelia | M | 2.3, 1.4 |
| 2.5 | 200-concurrent soak test | Murat + Amelia | M | 2.4, 0.3 |

### Epic 3 (Sprint 3): Resume + live decision UX (4 stories)

| Story | Title | Owner | Effort | Dependencies |
|---|---|---|---|---|
| 3.1 | Resume modal (5-row matrix) | Amelia + Sally | M | 2.4, 1.3 |
| 3.2 | VOD detour mode in player | Amelia | M | 3.1 |
| 3.3 | Session age + resume edge cases | Amelia | S | 3.1 |
| 3.4 | `POST /api/events` (204-always) | Amelia | M | 3.1, 3.2 |

### Epic 4 (Sprint 4): Polish + demo readiness (6 stories)

| Story | Title | Owner | Effort | Dependencies |
|---|---|---|---|---|
| 4.1 | Error states + retry | Amelia + Sally | M | 1.4, 2.4 |
| 4.2 | Loading skeletons | Sally + Amelia | M | 2.4, 4.1 |
| 4.3 | Mobile responsiveness | Amelia + Sally | M | 2.4, 3.1, 3.2, 4.1, 4.2 |
| 4.4 | 500-concurrent load test (G3 gate) | Murat + Amelia | M | 2.5, 0.3, all prior |
| 4.5 | Keyboard nav full coverage | Amelia + Sally | S | 2.3, 2.4, 3.1, 3.2 |
| 4.6 | Internal demo flow walkthrough doc | Sophia + Paige | S | all prior Sprint 4 |

---

## Acceptance Gates Trace (pivot §3 G1–G5)

| Gate | Description | Owning story | Validation |
|---|---|---|---|
| G1 | Cold load ≤2.5s p95 to first frame | 1.5 (baseline) + 4.4 (sustained) | 10-run p95 measured; report committed |
| G2 | Channel switch ≤500ms | 2.4 + 2.5 | Switch latency assertion in Playwright + k6 |
| G3 | 500 concurrent ±2s sync | 4.4 | 30-min k6 + parallel Playwright sync test |
| G4 | Return / resume correctness | 3.1 + 3.2 + 3.3 | 5 Playwright tests = 5 matrix rows |
| G5 | Multi-channel demo-quality | 2.4 + 4.x polish | 4.6 walkthrough doc dry-run |

All five gates have a clear owning story and a measurement method. No gate is orphaned.

---

## Risk Register Status (pivot §16 → epic-tv-1.md live mirror)

| ID | Status (now) | Owning story | Closure trigger |
|---|---|---|---|
| R1 | Open / High | 0.2 + 0.3 | Pipeline + infra both close |
| R2 | Open | 1.4 (two paths) + 4.3 (mobile cross-browser) | 4.3 DoD |
| R3 | Open | 1.3 + 1.4 | 1.3 + 1.4 DoD |
| R4 | Open (deferred) | (post-POC, D11) | Out of scope this epic |
| R5 | Open / High | 0.1 | 0.1 DoD |
| R6 | Closed | 1.3 + 3.3 | Already closed; 3.3 enforces |
| R7 | Open | 2.1 (editorial mix) | Post-demo revisit |
| R8 | Closed | (pivot §12.3 documented) | 1.4 enforces |
| R9 | Open (deferred per §11.5 state 5) | 4.1 (documents the gap) | Post-POC |
| R10 | Closed | 1.4 (`muted` autoplay) | 1.4 DoD |

No High-status risks are unowned. All Open risks have a closure path.

---

## Real-World Prerequisites (not story-quality issues — execution)

These remain pending because they require Chris/Vik human action; spec-quality is complete:

1. **Story 0.1 — Asset inventory.** Chris names 5 source MP4/MOV titles; Vik signs off rights per asset. §17.3 closes upon completion.
2. **Story 0.2 — Encoding pipeline.** Chris encodes Channel 1's first asset to HLS in R2 (Sprint 1 day 5 deadline per D16); 4 more by Sprint 2 day 5.
3. **Story 0.3 — Infrastructure.** Chris provisions Cloudflare account, Workers project, Neon project + dev branch, DNS records. ~half-day of work.

**Sprint 1 cannot start dev work until all three close.** They can run in parallel. Owners are named; deadlines are explicit.

---

## Recommended Next Steps

1. **Chris:** Block ~4 hours this week to close 0.3 (Cloudflare + Neon + DNS). Lowest-risk task; pure operational.
2. **Chris + Vik:** Confirm 5 asset titles + rights sign-offs (Story 0.1). Vik's portion is self-attestation per resolved §17.3 stance.
3. **Chris:** Run first asset through FFmpeg → R2 (Story 0.2) once 0.3's R2 bucket is live.
4. **Amelia:** Once 0.1 + 0.2 + 0.3 are green, kick off Story 1.1. Sprint 1 sprint plan from there is linear: 1.1 → 1.2 → 1.3 → 1.4 → 1.5.
5. **Optional / Mary's caveat from party-mode:** At Epic TV-1 close, author `_bmad-output/planning-artifacts/session-schema-changelog.md` consolidating SessionV1 → SessionV2 evolution. Adopted as non-blocking enhancement.

---

## Files Changed by This Review (2026-05-12)

| Path | Change |
|---|---|
| `_bmad-output/implementation-artifacts/stories/0-3-infrastructure-provisioning.md` | **Created** (Story 0.3 — Chris-owned infra provisioning, 20 ACs) |
| `_bmad-output/implementation-artifacts/stories/1-3-useservertime-and-usesession-hooks.md` | Added forward-compat schema note for D2/D3 |
| `_bmad-output/implementation-artifacts/stories/1-4-channelplayer-minimal-viable.md` | AC 23 extended to write `lastAssetHlsUrl` + `lastAssetTitle` (D2) |
| `_bmad-output/implementation-artifacts/stories/2-4-watch-slug-route-and-player-wiring.md` | AC 8 extended (D2); impl-note typo fixed |
| `_bmad-output/implementation-artifacts/stories/3-1-resume-modal.md` | AC 11 + AC 12 + impl notes updated for session-canonical (D2); schema additions called out (D3) |
| `_bmad-output/implementation-artifacts/stories/3-2-vod-detour-mode-in-player.md` | PlayerMode union extended with `hlsUrl` + `title`; AC 3, 4, 5 updated; impl notes resolved |
| `_bmad-output/implementation-artifacts/stories/3-3-session-age-and-edge-case-handling.md` | AC 5 + 7 updated for session-canonical; impl note added |
| All 23 story files | `status: draft` → `status: ready-for-dev` in frontmatter |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | 0-3 entry added; blockers list updated; readiness-review decisions noted |
| `_bmad-output/planning-artifacts/epic-tv-1.md` | Epic 0 section: Story 0.3 added; readiness-review note added |

---

## Verdict

**The plan is implementation-ready.**

22 original stories were already of unusually high quality. The 3 critical issues (no infra story, ambiguous VOD URL source, schema drift) and 2 fix-and-go issues (status mismatch, typo) are all resolved. Stories now form a clean DAG with all 5 acceptance gates traceable to owning stories.

The remaining blockers (0.1 rights diligence, 0.2 encoding, 0.3 provisioning) are real-world operational tasks for Chris and Vik — not planning gaps. Once those close, Amelia can run Sprint 1 in linear order with full confidence the spec answers every "what about X?" question.

— Implementation-Readiness assessor, 2026-05-12
