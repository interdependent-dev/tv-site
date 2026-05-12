---
story_id: 0.1
story_key: 0-1-confirm-asset-inventory
epic: 0
sprint: Pre-Sprint-1 Prep
title: Confirm asset inventory and close §17.3 gate
status: ready-for-dev
priority: Must (blocker for Sprint 1)
owners:
  - Vik (rights diligence)
  - Chris (content selection)
effort: S
dependencies: []
blocks: ["1.1", "2.1"]
risks: ["R1", "R5"]
project_context_refs: ["Cat 6 §17 table", "Cat 7 #18", "Cat 7 #21"]
pivot_doc_refs: ["§17.3", "§16 R1", "§16 R5"]
decisions_resolved: ["D2", "D4", "D5", "D6", "D7", "D8"]
---

# Story 0.1: Confirm Asset Inventory and Close §17.3 Gate

## User Story

As **Vik (rights diligence owner)** and **Chris (content selector)**,
I want a confirmed list of 5 source assets with documented rights and editorial classification,
So that Sprint 1's data model has a deterministic content foundation and §17.3 (the only remaining gated §17 decision) closes.

## Context

§17.3 is the last §17 decision blocking Sprint 1 story authorship (per project-context.md Cat 6 §17 table + Cat 7 #18). Six §17 decisions Closed 2026-05-12; 17.3 remains Gated until this story closes.

Resolution simplified by the 2026-05-12 MVP triage (D2): content is Chris's own MP4/MOV + pre-screening films he owns. No licensed third-party content in POC scope. This deflates Murat's "lawsuit trap" framing of D6/D7 because Chris owns the content end-to-end — Vik's diligence is self-attestation against existing paperwork, not external counsel review.

## Decisions Already Resolved (2026-05-12 party-mode triage)

| ID | Decision | Resolution |
|---|---|---|
| D2 | 5-asset shortlist | Chris's owned MP4/MOV + pre-screening films |
| D4 | Channel 1 archetype | Ambient always-on (invite-to-linger) |
| D5 | Fallback if <5 ready | Ship N (N≥3), loop to fill, document, do not relitigate |
| D6 | Territorial rights | Vik's existing paperwork + written sign-off per asset |
| D7 | Music streaming clearance | Vik's music rights records + written sign-off per asset |
| D8 | SAG-AFTRA status | Self-attestation: confirm non-signatory OR document exposure |

## Acceptance Criteria

1. **Chris names 5 source assets.** Title, runtime in seconds, file format (MP4/MOV), current storage location captured in a project-tracker artifact.
2. **Channel 1 selected.** One asset/channel is designated as ambient-always-on default (D4). Documented in the tracker.
3. **Per-asset rights sign-off recorded.** For each of the 5 assets, Vik captures in writing:
   - Worldwide rights confirmation (no territorial carve-outs) OR documented territorial exposure.
   - Music sync + streaming clearance confirmation OR documented gaps.
   - SAG-AFTRA non-signatory status OR documented exposure with mitigation plan.
4. **Fallback path executed if <5 viable.** If fewer than 5 assets pass the diligence + selection step, scope falls to N≥3 channels per D5; the count is recorded; no debate.
5. **§17.3 status updated.** `_bmad-output/project-context.md` Cat 6 §17 table: 17.3 moves "Gated" → "Closed 2026-05-12 (or actual close date)".
6. **R1 + R5 status updated** in the epic file's live risk register: from "Open / High" to "Closed" once AC 1–5 are met.

## Acceptance Tests / Verification

**Manual verification checklist (Vik + Chris sign-off):**

- [ ] Tracker artifact (Google Doc / Notion / repo doc) exists with all 5 assets listed.
- [ ] Each asset has a tracker row with: title, runtime_sec, format, storage_location, rights_status, music_status, sag_status, channel_assignment.
- [ ] Channel 1 row has `is_default: true`.
- [ ] project-context.md frontmatter has §17.3 in `decisions_resolved: Closed`.
- [ ] Epic file risk register R1 + R5 status = Closed.

## Dependencies

- **Upstream:** None — this story is the unblocker.
- **Downstream (blocks):**
  - Story 1.1 (Data model + seed) requires Channel 1's asset for seed.
  - Story 2.1 (Seed 4 additional channels) requires Channels 2–5's assets.

## Risk Alignment (pivot §16)

| ID | Risk | Status going in | Status this story exits | Mitigation |
|---|---|---|---|---|
| R1 | Asset encoding pipeline absent | Open / High | This story confirms inventory; Story 0.2 closes the pipeline. R1 deflates to Closed when both 0.1 + 0.2 close. | This story enumerates what to encode; 0.2 establishes how. |
| R5 | Content licensing unclear | Open / High | Closed | D2 resolution (Chris owns content) + Vik's per-asset attestation. |
| R4 | Accessibility (CC, audio desc) deferred | Open | Open (deferred per D11) | Document the gap in the tracker per asset; not a close-blocker. |

## Implementation Notes

- The tracker artifact can be a Google Doc, Notion page, or `_bmad-output/planning-artifacts/asset-inventory.md` in the repo. Recommend in-repo for grep-ability.
- Vik's sign-off can be as light as "I, Vik, confirm worldwide rights / music streaming clearance / non-signatory status for [asset title] based on existing paperwork dated [date]." This is self-attestation, NOT a legal opinion. The diligence weight is appropriate for staging-scoped POC content owned by Chris (per the §17 party-mode resolution).
- If any asset fails diligence, it gets cut and Channel N is reassigned OR the count drops to 4. No public-domain placeholder substitution per resolved §17.3 stance.

## Definition of Done

- [ ] AC 1–6 met.
- [ ] project-context.md frontmatter + Cat 6 §17 table updated.
- [ ] Epic file risk register R1+R5 status reflects closure.
- [ ] Sprint-status.yaml story `0-1-confirm-asset-inventory` flips `backlog → ready-for-dev → done`.
- [ ] Story 1.1 and Story 0.2 unblocked.
