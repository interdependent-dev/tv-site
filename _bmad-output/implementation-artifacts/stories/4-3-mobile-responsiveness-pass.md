---
story_id: 4.3
story_key: 4-3-mobile-responsiveness-pass
epic: 4
sprint: Sprint 4 — Polish and demo readiness
title: Mobile responsiveness pass (iOS Safari + Android Chrome)
status: ready-for-dev
priority: Must
owners: ["Amelia (impl) + Sally (UX)"]
effort: M
dependencies: ["2.4", "3.1", "3.2", "4.1", "4.2"]
blocks: []
risks: ["R2", "R9"]
project_context_refs:
  - "Cat 1 (hls.js + native HLS distinct paths)"
  - "Cat 7 #13 (two code paths — Safari native HLS)"
pivot_doc_refs: ["§11 (UX)", "Sprint 4 Story 4.3"]
---

# Story 4.3: Mobile Responsiveness Pass

## User Story

As **a mobile viewer**,
I want full functionality across iPhone Safari and Android Chrome in portrait AND landscape,
So that the POC demo works on the most likely first-impression device — a phone.

## Context

Two distinct player code paths are both exercised on mobile:
- **iOS Safari**: native HLS playback path (Cat 7 #13).
- **Android Chrome**: hls.js path.

Mobile-specific UI considerations: thumb-reach tile sizes, swipe-to-reveal guide, ≥44pt touch targets, no hover states.

## Acceptance Criteria

1. **iPhone (iOS 17+) Safari full functionality:**
   - Player autoplays muted, tap-to-unmute works.
   - Native HLS path: `loadedmetadata` → `currentTime` → `play()` sequence verified.
   - Channel guide bottom-swipe-up gesture reveals.
   - Resume modal displays and is fully interactive.
   - VOD detour works including scrubber.
2. **Android Chrome full functionality:**
   - Same as iPhone with hls.js path.
   - `MANIFEST_PARSED` seek sequence verified.
3. **Portrait AND landscape both work** on both platforms:
   - Player resizes correctly on orientation change.
   - Guide layout adapts (vertical stack in portrait, horizontal row in landscape).
   - hls.js continues playback across orientation changes (no re-init).
4. **Touch-target sizes ≥ 44pt** (iOS HIG; Android Material recommends ≥48dp ≈ same): all buttons, modal CTAs, guide tiles' interactive zones.
5. **No hover-only states** — every hover effect has an equivalent active/focus state for touch.
6. **`safe-area-inset-*` respected** for notched devices (iPhone X+). Player chrome doesn't hide behind the notch.
7. **`viewport` meta tag**: `width=device-width, initial-scale=1, viewport-fit=cover` (matches legacy `index.html`).
8. **Picture-in-picture (PiP) enabled by default** (pivot §18 Q6). Native browser PiP, no custom UI.

## Acceptance Tests

**Playwright (mobile emulation):**
- [ ] iPhone 14 Safari: cold load → autoplay (muted) → tap-unmute → playback continues.
- [ ] Pixel 7 Chrome: same flow.
- [ ] Orientation change: rotate from portrait to landscape; assert playback continues; assert layout updates.
- [ ] Touch-target audit: assert all interactive elements have computed width AND height ≥ 44pt (use Playwright's `boundingBox()`).
- [ ] Safari native HLS test (real device or BrowserStack): `<video>` plays HLS without hls.js loading.

**Manual:**
- [ ] Real iPhone test session (one device, one Vik / Chris / team member).
- [ ] Real Android test session.

## Dependencies

- **Upstream:** 2.4, 3.1, 3.2, 4.1, 4.2 (all UI surfaces need to exist before mobile polish).
- **Downstream:** none.

## Risk Alignment (pivot §16)

| ID | Risk | Mitigation |
|---|---|---|
| R2 | hls.js / Safari behavioral diffs | This story IS the closure. R2 → Closed when AC 1–8 met. |
| R9 | Mobile data overage on cellular | DEFERRED per resolved §11.5 (state 5). Bitrate cap on cellular = post-POC. Document in known-gaps.md. |

## Implementation Notes

- **Safari native HLS detection** (Cat 7 #13):
  ```ts
  const supportsNativeHls = video.canPlayType('application/vnd.apple.mpegurl') !== '';
  ```
  Use this in `<ChannelPlayer>` to branch. NOT `navigator.userAgent` — false positives on iPad Chrome (which still uses WebKit / native HLS).
- **Swipe-to-reveal guide on mobile:** Pointer Events API. Track `pointerdown` y-coordinate at bottom edge of viewport; track `pointermove` delta. If delta > threshold (60px), reveal guide.
- **PiP:** add `controlslist="..."` to `<video>` to expose the native control. iOS Safari includes the PiP button automatically. Android Chrome since v96 also supports PiP via the native control.
- **Safe area insets:**
  ```css
  padding-top: env(safe-area-inset-top);
  padding-bottom: env(safe-area-inset-bottom);
  ```
  Apply to player chrome and guide overlay.

## Definition of Done

- [ ] AC 1–8 met.
- [ ] Playwright mobile emulation tests green.
- [ ] Real-device manual test session completed (notes in story close-out).
- [ ] R2 status = Closed in epic risk register.
- [ ] Sprint-status.yaml `4-3-mobile-responsiveness-pass` → done.
