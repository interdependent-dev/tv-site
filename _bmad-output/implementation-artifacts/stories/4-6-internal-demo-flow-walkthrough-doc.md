---
story_id: 4.6
story_key: 4-6-internal-demo-flow-walkthrough-doc
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Internal demo flow walkthrough doc
status: ready-for-dev
priority: Should
owners: ["Sophia (narrative) + Paige (tech writing)"]
effort: S
dependencies: ["all prior Sprint 4 stories"]
blocks: []
risks: []
project_context_refs: ["Cat 6 (documentation discipline — narrative doc reserved for demo)"]
pivot_doc_refs: ["§3 (acceptance gates G1–G5)", "Sprint 4 Story 4.6"]
---

# Story 4.6: Internal Demo Flow Walkthrough Doc

## User Story

As **a team member showing INTERDEPENDENT.TV to external stakeholders** (LPs, prospective creators, partners),
I want a scripted demo flow covering all five §3 acceptance gates,
So that the demo lands consistently and intentionally — and the platform's distinctive thesis comes through, not just "look, video plays."

## Context

This is the only narrative doc allowed for external stakeholders per project-context.md Cat 6. Authored by Sophia in collaboration with Paige. The doc IS the demo's spec; if it works for someone reading the doc, it works for the team.

## Acceptance Criteria

1. **Doc at `_bmad-output/planning-artifacts/demo-flow-walkthrough.md`** (or similar). Markdown, ~3–5 pages.
2. **Structure:**
   - **Opening (60s):** Context for the stakeholder. What INTERDEPENDENT.TV is. Why it matters to the platform thesis (the front door for the broadest audience; the "TV is on when you open it" mental model).
   - **Demo segment 1 — Cold load (G1):** Open `interdependent.tv` in a fresh browser. Highlight: channel plays in <2.5s. Why this matters: anonymous, zero-friction; the FAST mechanism in action.
   - **Demo segment 2 — Channel switching (G2):** Open guide, switch channels. Highlight: ≤500ms; same player; the editorial range across the 5 channels.
   - **Demo segment 3 — Concurrent sync (G3):** Open the same channel in a second window. Highlight: ±2s sync, no server-side coordination. The architectural punchline (every viewer's browser does the math).
   - **Demo segment 4 — Return + resume (G4 + G5):** Refresh after watching for a bit. Highlight: Resume Modal Row 1/2/3. The VOD detour. The fact that "continue where you left off" works for LIVE TV — a behavior no FAST competitor offers.
   - **Closing (60s):** Architectural callout — what's not here that's also not needed (no live encode, no DRM, no auth gate). What's NEXT (scale sprint §14, true-live for ceremony events, federated creator channel post-POC).
3. **For each segment:**
   - The exact narration to deliver (or a tight outline if the speaker is improv-friendly).
   - The exact clicks / interactions to perform.
   - The exact metric / behavior to call out.
   - What to do if something breaks (rollback narrative — never apologize on stage; pivot to the next segment).
4. **Pre-demo checklist:** browser cache cleared; preview URL warm (one navigation 5 min before); k6 NOT running concurrently (would skew p95); device + projector compatibility verified.
5. **Post-demo Q&A primer:** 3–5 likely questions with crisp answers ("How does this scale to true live?" → reference §14.4 + the architectural separation between channel modes).

## Acceptance Tests

- [ ] Team member who was NOT involved in the project does a dry run from the doc. Catches gaps. Doc revised. Repeat until clean.
- [ ] Live dry-run on a real projector + actual demo URL.

## Dependencies

- **Upstream:** All Sprint 4 stories (all must be working).
- **Downstream:** none.

## Risk Alignment

(No §16 risk; communication artifact.)

## Implementation Notes

- **Tone:** confident, restrained, technical-when-it-matters. Avoid hype. The architecture IS the differentiator — let it speak.
- **Length:** if a stakeholder cannot read this doc in 10 minutes, it's too long.
- **Audience tiers:** the doc may be shown to (a) LPs who think in metrics + roadmap, (b) prospective creators who think in audience + distribution, (c) engineers doing tech diligence. One doc; the speaker chooses emphasis.
- **Sophia's voice** (per BMAD persona): narrative arcs grounded in human truth. The story is "TV is back, but it actually understands you. Look at this moment."
- **Paige's role:** precision pass on technical claims; ensure every metric cited is verifiable; cross-link to baseline / load-test reports from 1.5 and 4.4.

## Definition of Done

- [ ] AC 1–5 met.
- [ ] Dry run by uninvolved team member produces no surprises.
- [ ] Demo URL warm-up procedure documented.
- [ ] Sprint-status.yaml `4-6-internal-demo-flow-walkthrough-doc` → done.
- [ ] Epic TV-1 DoD criterion (demo-ready) satisfied → Epic moves to `done`.
