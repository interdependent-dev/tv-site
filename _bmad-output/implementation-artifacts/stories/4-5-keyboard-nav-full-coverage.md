---
story_id: 4.5
story_key: 4-5-keyboard-nav-full-coverage
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Keyboard navigation full coverage (pivot §11.6 table)
status: ready-for-dev
priority: Should
owners: ["Amelia (impl) + Sally (UX review)"]
effort: S
dependencies: ["2.3", "2.4", "3.1", "3.2"]
blocks: []
risks: []
project_context_refs: []
pivot_doc_refs: ["§11.6 (keyboard nav table)"]
---

# Story 4.5: Keyboard Navigation Full Coverage

## User Story

As **a desktop / TV-browser viewer using keyboard control**,
I want every interaction reachable by keyboard shortcuts matching pivot §11.6,
So that the experience feels TV-native — and the demo on a laptop projector works without a touchpad.

## Context

Pivot §11.6 keybinding table is the spec. Story 2.3 implemented MVP keyboard nav (↑↓, Enter, 1–5). This story closes the remaining bindings.

## Acceptance Criteria

Implement the full pivot §11.6 table:

1. **`Space` / `K`** in player context: play/pause toggle.
2. **`M`** in player context: mute toggle.
3. **`F`** in player context: fullscreen toggle (via `videoEl.requestFullscreen()` / `document.exitFullscreen()`).
4. **`G` / `↓`** in player context: open guide.
5. **`Esc`** in guide context: close guide.
6. **`↑↓`** in guide context: channel up/down.
7. **`←→`** in guide-EPG context: earlier/later time slot. **NOTE:** EPG is DEFERRED per pivot §11.3; this binding has no effect in POC but should not break.
8. **`Enter`** in guide context: switch to highlighted channel.
9. **`1`–`9`** in player context: jump to channel by sort order (TV-style; only 1–5 active in POC since 5 channels).

### Context awareness

10. Bindings scoped to current focus context. Pressing `M` while focus is in an `<input>` does NOT mute the player.
11. Global keybinding handler at `app/(tv)/_components/KeyboardController.tsx` (or similar). Subscribes via `document.addEventListener('keydown', ...)`; cleans up on unmount.
12. Skip handler if `document.activeElement.tagName === 'INPUT' || 'TEXTAREA' || isContentEditable`.

### Discoverability

13. Optional: a help overlay (`?` key) listing keybindings. NICE-TO-HAVE; can defer to known-gaps if scope tight.

## Acceptance Tests

**Playwright:**
- [ ] All 9 bindings tested via `page.keyboard.press(...)` from appropriate context; assert expected state change.
- [ ] `M` while focused on a search input does NOT mute player.

## Dependencies

- **Upstream:** 2.3 (basic guide kb), 2.4 (slug routing), 3.1+3.2 (modal + player modes).
- **Downstream:** none.

## Risk Alignment

(No §16 risk; accessibility / polish.)

## Implementation Notes

- **Centralized handler reasoning:** distributing keybindings across components leads to conflicts. One handler with context-routing logic keeps it sane.
- **Fullscreen quirk:** iOS Safari does NOT support `requestFullscreen()` on regular elements — only `<video>` and only via `webkitEnterFullscreen()`. Branch accordingly.
- **`?` help overlay:** if shipped, a simple modal listing the table. Toggleable. Style consistent with brand frame.

## Definition of Done

- [ ] AC 1–12 met (AC 13 optional).
- [ ] Playwright tests for all bindings.
- [ ] No conflicts when typing in inputs.
- [ ] Sprint-status.yaml `4-5-keyboard-nav-full-coverage` → done.
